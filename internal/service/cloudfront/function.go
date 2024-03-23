// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloudfront

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	awstypes "github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/route53domains/types"

	// "github.com/hashicorp/aws-sdk-go-base/v2/awsv1shim/v2/tfawserr"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

// @SDKResource("aws_cloudfront_function")
func ResourceFunction() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceFunctionCreate,
		ReadWithoutTimeout:   resourceFunctionRead,
		UpdateWithoutTimeout: resourceFunctionUpdate,
		DeleteWithoutTimeout: resourceFunctionDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"code": {
				Type:     schema.TypeString,
				Required: true,
			},
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"live_stage_etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"publish": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"runtime": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: func(val interface{}, key string) ([]string, []error) {
					runtimeValues := awstypes.FunctionRuntime.Values(awstypes.FunctionRuntime(key))
					runtimes := make([]string, len(runtimeValues))
					for i, rt := range runtimeValues {
						runtimes[i] = string(rt)
					}
					return validation.StringInSlice(runtimes, false)(val, key)
				},
			},

			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceFunctionCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	functionName := d.Get("name").(string)
	input := &cloudfront.CreateFunctionInput{
		FunctionCode: []byte(d.Get("code").(string)),
		FunctionConfig: &awstypes.FunctionConfig{
			Comment: aws.String(d.Get("comment").(string)),
			Runtime: awstypes.FunctionRuntime(d.Get("runtime").(string)),
		},
		Name: aws.String(functionName),
	}

	log.Printf("[DEBUG] Creating CloudFront Function: %s", functionName)
	output, err := conn.CreateFunction(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating CloudFront Function (%s): %s", functionName, err)
	}

	d.SetId(aws.ToString(output.FunctionSummary.Name))

	if d.Get("publish").(bool) {
		input := &cloudfront.PublishFunctionInput{
			Name:    aws.String(d.Id()),
			IfMatch: output.ETag,
		}

		log.Printf("[DEBUG] Publishing CloudFront Function: %s", input)
		_, err := conn.PublishFunction(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "publishing CloudFront Function (%s): %s", d.Id(), err)
		}
	}

	return append(diags, resourceFunctionRead(ctx, d, meta)...)
}

func resourceFunctionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	describeFunctionOutput, err := FindFunctionByNameAndStage(ctx, conn, d.Id(), awstypes.FunctionStageDevelopment)

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] CloudFront Function (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading CloudFront Function (%s) DEVELOPMENT stage: %s", d.Id(), err)
	}

	d.Set("arn", describeFunctionOutput.FunctionSummary.FunctionMetadata.FunctionARN)
	d.Set("comment", describeFunctionOutput.FunctionSummary.FunctionConfig.Comment)
	d.Set("etag", describeFunctionOutput.ETag)
	d.Set("name", describeFunctionOutput.FunctionSummary.Name)
	d.Set("runtime", describeFunctionOutput.FunctionSummary.FunctionConfig.Runtime)
	d.Set("status", describeFunctionOutput.FunctionSummary.Status)

	getFunctionOutput, err := conn.GetFunction(ctx, &cloudfront.GetFunctionInput{
		Name:  aws.String(d.Id()),
		Stage: awstypes.FunctionStageDevelopment,
	})

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading CloudFront Function (%s) DEVELOPMENT stage code: %s", d.Id(), err)
	}

	d.Set("code", string(getFunctionOutput.FunctionCode))

	describeFunctionOutput, err = FindFunctionByNameAndStage(ctx, conn, d.Id(), awstypes.FunctionStageLive)

	if tfresource.NotFound(err) {
		d.Set("live_stage_etag", "")
	} else if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading CloudFront Function (%s) LIVE stage: %s", d.Id(), err)
	} else {
		d.Set("live_stage_etag", describeFunctionOutput.ETag)
	}

	return diags
}

func resourceFunctionUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)
	etag := d.Get("etag").(string)

	if d.HasChanges("code", "comment", "runtime") {
		input := &cloudfront.UpdateFunctionInput{
			FunctionCode: []byte(d.Get("code").(string)),
			FunctionConfig: &awstypes.FunctionConfig{
				Comment: aws.String(d.Get("comment").(string)),
				Runtime: awstypes.FunctionRuntime(d.Get("runtime").(string)),
			},
			Name:    aws.String(d.Id()),
			IfMatch: aws.String(etag),
		}

		log.Printf("[INFO] Updating CloudFront Function: %s", d.Id())
		output, err := conn.UpdateFunction(ctx, input)
		if err != nil {
			return sdkdiag.AppendErrorf(diags, "updating CloudFront Function (%s): %s", d.Id(), err)
		}

		etag = aws.ToString(output.ETag)
	}

	if d.Get("publish").(bool) {
		input := &cloudfront.PublishFunctionInput{
			Name:    aws.String(d.Id()),
			IfMatch: aws.String(etag),
		}

		log.Printf("[DEBUG] Publishing CloudFront Function: %s", d.Id())
		_, err := conn.PublishFunction(ctx, input)

		if err != nil {
			return sdkdiag.AppendErrorf(diags, "publishing CloudFront Function (%s): %s", d.Id(), err)
		}
	}

	return append(diags, resourceFunctionRead(ctx, d, meta)...)
}

func resourceFunctionDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	log.Printf("[INFO] Deleting CloudFront Function: %s", d.Id())
	_, err := conn.DeleteFunction(ctx, &cloudfront.DeleteFunctionInput{
		Name:    aws.String(d.Id()),
		IfMatch: aws.String(d.Get("etag").(string)),
	})

	if errs.IsAErrorMessageContains[*types.InvalidInput](err, "not found") {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting CloudFront Function (%s): %s", d.Id(), err)
	}

	return diags
}

type invalidParameterValueError struct {
	error
}
