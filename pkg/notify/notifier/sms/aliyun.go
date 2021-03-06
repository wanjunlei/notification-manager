package sms

import (
	"context"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	dysmsapi "github.com/alibabacloud-go/dysmsapi-20170525/v2/client"
	"github.com/kubesphere/notification-manager/pkg/apis/v2beta2"
	"github.com/kubesphere/notification-manager/pkg/notify/config"
)

const (
	aliyunMaxPhoneNums = 1000
)

type AliyunNotifier struct {
	SignName        string
	NotifierCfg     *config.Config
	TemplateCode    string
	AccessKeyId     *v2beta2.Credential
	AccessKeySecret *v2beta2.Credential
	PhoneNums       string
}

func NewAliyunProvider(c *config.Config, providers *v2beta2.Providers, phoneNumbers []string) Provider {
	phoneNums := handleAliyunPhoneNums(phoneNumbers)
	return &AliyunNotifier{
		SignName:        providers.Aliyun.SignName,
		NotifierCfg:     c,
		TemplateCode:    providers.Aliyun.TemplateCode,
		AccessKeyId:     providers.Aliyun.AccessKeyId,
		AccessKeySecret: providers.Aliyun.AccessKeySecret,
		PhoneNums:       phoneNums,
	}
}

func (a *AliyunNotifier) MakeRequest(ctx context.Context, messages string) error {
	accessKeyId, err := a.NotifierCfg.GetCredential(a.AccessKeyId)
	if err != nil {
		return fmt.Errorf("[Aliyun  SendSms] cannot get accessKeyId: %s", err.Error())
	}
	accessKeySecret, err := a.NotifierCfg.GetCredential(a.AccessKeySecret)
	if err != nil {
		return fmt.Errorf("[Aliyun  SendSms] cannot get accessKeySecret: %s", err.Error())
	}
	config := &openapi.Config{}
	config.AccessKeyId = &accessKeyId
	config.AccessKeySecret = &accessKeySecret
	client, err := dysmsapi.NewClient(config)
	if err != nil {
		return fmt.Errorf("[Aliyun  SendSms] cannot make a client with accessKeyId:%s,accessKeySecret:%s",
			a.AccessKeyId.ValueFrom.SecretKeyRef.Name, a.AccessKeySecret.ValueFrom.SecretKeyRef.Name)
	}

	templateParam := `{"code":"` + messages + `"}`
	req := &dysmsapi.SendSmsRequest{
		PhoneNumbers:  &a.PhoneNums,
		SignName:      &a.SignName,
		TemplateCode:  &a.TemplateCode,
		TemplateParam: &templateParam,
	}
	resp, err := client.SendSms(req)
	if err != nil {
		return fmt.Errorf("[Aliyun  SendSms] An API error occurs: %s", err.Error())
	}

	if stringValue(resp.Body.Code) != "OK" {
		return fmt.Errorf("[Aliyun  SendSms] Send failed: %s", fmt.Errorf(stringValue(resp.Body.Message)))
	}

	return nil
}

func handleAliyunPhoneNums(phoneNumbers []string) string {
	if len(phoneNumbers) > aliyunMaxPhoneNums {
		phoneNumbers = phoneNumbers[:aliyunMaxPhoneNums]
	}
	return strings.Join(phoneNumbers, ",")
}
