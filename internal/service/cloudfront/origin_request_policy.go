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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/sdkdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
)

// @SDKResource("aws_cloudfront_origin_request_policy")
func ResourceOriginRequestPolicy() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceOriginRequestPolicyCreate,
		ReadWithoutTimeout:   resourceOriginRequestPolicyRead,
		UpdateWithoutTimeout: resourceOriginRequestPolicyUpdate,
		DeleteWithoutTimeout: resourceOriginRequestPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"comment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"cookies_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cookie_behavior": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(val interface{}, key string) ([]string, []error) {
								runtimeValues := awstypes.OriginRequestPolicyCookieBehavior.Values(awstypes.OriginRequestPolicyCookieBehavior(key))
								runtimes := make([]string, len(runtimeValues))
								for i, rt := range runtimeValues {
									runtimes[i] = string(rt)
								}
								return validation.StringInSlice(runtimes, false)(val, key)
							},
						},
						"cookies": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"items": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
			"etag": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"headers_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"header_behavior": {
							Type:     schema.TypeString,
							Optional: true,
							ValidateFunc: func(val interface{}, key string) ([]string, []error) {
								runtimeValues := awstypes.OriginRequestPolicyHeaderBehavior.Values(awstypes.OriginRequestPolicyHeaderBehavior(key))
								runtimes := make([]string, len(runtimeValues))
								for i, rt := range runtimeValues {
									runtimes[i] = string(rt)
								}
								return validation.StringInSlice(runtimes, false)(val, key)
							},
						},
						"headers": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"items": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"query_strings_config": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"query_string_behavior": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(val interface{}, key string) ([]string, []error) {
								runtimeValues := awstypes.OriginRequestPolicyQueryStringBehavior.Values(awstypes.OriginRequestPolicyQueryStringBehavior(key))
								runtimes := make([]string, len(runtimeValues))
								for i, rt := range runtimeValues {
									runtimes[i] = string(rt)
								}
								return validation.StringInSlice(runtimes, false)(val, key)
							},
						},
						"query_strings": {
							Type:     schema.TypeList,
							MaxItems: 1,
							Optional: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"items": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func resourceOriginRequestPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	name := d.Get("name").(string)
	apiObject := &awstypes.OriginRequestPolicyConfig{
		Name: aws.String(name),
	}

	if v, ok := d.GetOk("comment"); ok {
		apiObject.Comment = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cookies_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.CookiesConfig = expandOriginRequestPolicyCookiesConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("headers_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.HeadersConfig = expandOriginRequestPolicyHeadersConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("query_strings_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.QueryStringsConfig = expandOriginRequestPolicyQueryStringsConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	input := &cloudfront.CreateOriginRequestPolicyInput{
		OriginRequestPolicyConfig: apiObject,
	}

	log.Printf("[DEBUG] Creating CloudFront Origin Request Policy: (%s)", input)
	output, err := conn.CreateOriginRequestPolicy(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "creating CloudFront Origin Request Policy (%s): %s", name, err)
	}

	d.SetId(aws.ToString(output.OriginRequestPolicy.Id))

	return append(diags, resourceOriginRequestPolicyRead(ctx, d, meta)...)
}

func resourceOriginRequestPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	output, err := FindOriginRequestPolicyByID(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] CloudFront Origin Request Policy (%s) not found, removing from state", d.Id())
		d.SetId("")
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "reading CloudFront Origin Request Policy (%s): %s", d.Id(), err)
	}

	apiObject := output.OriginRequestPolicy.OriginRequestPolicyConfig
	d.Set("comment", apiObject.Comment)
	if apiObject.CookiesConfig != nil {
		if err := d.Set("cookies_config", []interface{}{flattenOriginRequestPolicyCookiesConfig(apiObject.CookiesConfig)}); err != nil {
			return sdkdiag.AppendErrorf(diags, "setting cookies_config: %s", err)
		}
	} else {
		d.Set("cookies_config", nil)
	}
	d.Set("etag", output.ETag)
	if apiObject.HeadersConfig != nil {
		if err := d.Set("headers_config", []interface{}{flattenOriginRequestPolicyHeadersConfig(apiObject.HeadersConfig)}); err != nil {
			return sdkdiag.AppendErrorf(diags, "setting headers_config: %s", err)
		}
	} else {
		d.Set("headers_config", nil)
	}
	d.Set("name", apiObject.Name)
	if apiObject.QueryStringsConfig != nil {
		if err := d.Set("query_strings_config", []interface{}{flattenOriginRequestPolicyQueryStringsConfig(apiObject.QueryStringsConfig)}); err != nil {
			return sdkdiag.AppendErrorf(diags, "setting query_strings_config: %s", err)
		}
	} else {
		d.Set("query_strings_config", nil)
	}

	return diags
}

func resourceOriginRequestPolicyUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	//
	// https://docs.aws.amazon.com/cloudfront/latest/APIReference/API_UpdateOriginRequestPolicy.html:
	// "When you update an origin request policy configuration, all the fields are updated with the values provided in the request. You cannot update some fields independent of others."
	//
	apiObject := &awstypes.OriginRequestPolicyConfig{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("comment"); ok {
		apiObject.Comment = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cookies_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.CookiesConfig = expandOriginRequestPolicyCookiesConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("headers_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.HeadersConfig = expandOriginRequestPolicyHeadersConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	if v, ok := d.GetOk("query_strings_config"); ok && len(v.([]interface{})) > 0 && v.([]interface{})[0] != nil {
		apiObject.QueryStringsConfig = expandOriginRequestPolicyQueryStringsConfig(v.([]interface{})[0].(map[string]interface{}))
	}

	input := &cloudfront.UpdateOriginRequestPolicyInput{
		Id:                        aws.String(d.Id()),
		IfMatch:                   aws.String(d.Get("etag").(string)),
		OriginRequestPolicyConfig: apiObject,
	}

	log.Printf("[DEBUG] Updating CloudFront Origin Request Policy: (%s)", input)
	_, err := conn.UpdateOriginRequestPolicy(ctx, input)

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "updating CloudFront Origin Request Policy (%s): %s", d.Id(), err)
	}

	return append(diags, resourceOriginRequestPolicyRead(ctx, d, meta)...)
}

func resourceOriginRequestPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CloudFrontClient(ctx)

	log.Printf("[DEBUG] Deleting CloudFront Origin Request Policy: (%s)", d.Id())
	_, err := conn.DeleteOriginRequestPolicy(ctx, &cloudfront.DeleteOriginRequestPolicyInput{
		Id:      aws.String(d.Id()),
		IfMatch: aws.String(d.Get("etag").(string)),
	})

	if errs.IsAErrorMessageContains[*types.InvalidInput](err, "not found") {
		return diags
	}

	if err != nil {
		return sdkdiag.AppendErrorf(diags, "deleting CloudFront Origin Request Policy (%s): %s", d.Id(), err)
	}

	return diags
}

func expandOriginRequestPolicyCookiesConfig(tfMap map[string]interface{}) *awstypes.OriginRequestPolicyCookiesConfig {
	if tfMap == nil {
		return nil
	}

	apiObject := &awstypes.OriginRequestPolicyCookiesConfig{}

	if v, ok := tfMap["cookie_behavior"].(string); ok && v != "" {
		apiObject.CookieBehavior = awstypes.OriginRequestPolicyCookieBehavior(v)
	}

	if v, ok := tfMap["cookies"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
		apiObject.Cookies = expandCookieNames(v[0].(map[string]interface{}))
	}

	return apiObject
}

func expandOriginRequestPolicyHeadersConfig(tfMap map[string]interface{}) *awstypes.OriginRequestPolicyHeadersConfig {
	if tfMap == nil {
		return nil
	}

	apiObject := &awstypes.OriginRequestPolicyHeadersConfig{}

	if v, ok := tfMap["header_behavior"].(string); ok && v != "" {
		apiObject.HeaderBehavior = awstypes.OriginRequestPolicyHeaderBehavior(v)
	}

	if v, ok := tfMap["headers"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
		apiObject.Headers = expandHeaders(v[0].(map[string]interface{}))
	}

	return apiObject
}

func expandOriginRequestPolicyQueryStringsConfig(tfMap map[string]interface{}) *awstypes.OriginRequestPolicyQueryStringsConfig {
	if tfMap == nil {
		return nil
	}

	apiObject := &awstypes.OriginRequestPolicyQueryStringsConfig{}

	if v, ok := tfMap["query_string_behavior"].(string); ok && v != "" {
		apiObject.QueryStringBehavior = awstypes.OriginRequestPolicyQueryStringBehavior(v)
	}

	if v, ok := tfMap["query_strings"].([]interface{}); ok && len(v) > 0 && v[0] != nil {
		apiObject.QueryStrings = expandQueryStringNames(v[0].(map[string]interface{}))
	}

	return apiObject
}

func flattenOriginRequestPolicyCookiesConfig(apiObject *awstypes.OriginRequestPolicyCookiesConfig) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.CookieBehavior; v != awstypes.OriginRequestPolicyCookieBehavior("") {
		tfMap["cookie_behavior"] = awstypes.OriginRequestPolicyCookieBehavior(v)
	}

	if v := flattenCookieNames(apiObject.Cookies); len(v) > 0 {
		tfMap["cookies"] = []interface{}{v}
	}

	return tfMap
}

func flattenOriginRequestPolicyHeadersConfig(apiObject *awstypes.OriginRequestPolicyHeadersConfig) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.HeaderBehavior; v != awstypes.OriginRequestPolicyHeaderBehavior("") {
		tfMap["header_behavior"] = awstypes.OriginRequestPolicyHeaderBehavior(v)
	}

	if v := flattenHeaders(apiObject.Headers); len(v) > 0 {
		tfMap["headers"] = []interface{}{v}
	}

	return tfMap
}

func flattenOriginRequestPolicyQueryStringsConfig(apiObject *awstypes.OriginRequestPolicyQueryStringsConfig) map[string]interface{} {
	if apiObject == nil {
		return nil
	}

	tfMap := map[string]interface{}{}

	if v := apiObject.QueryStringBehavior; v != awstypes.OriginRequestPolicyQueryStringBehavior("") {
		tfMap["query_string_behavior"] = awstypes.OriginRequestPolicyQueryStringBehavior(v)
	}

	if v := flattenQueryStringNames(apiObject.QueryStrings); len(v) > 0 {
		tfMap["query_strings"] = []interface{}{v}
	}

	return tfMap
}
