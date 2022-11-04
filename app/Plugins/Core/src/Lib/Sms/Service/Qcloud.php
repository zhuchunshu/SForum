<?php

namespace App\Plugins\Core\src\Lib\Sms\Service;

use TencentCloud\Common\Credential;
use TencentCloud\Common\Profile\ClientProfile;
use TencentCloud\Common\Profile\HttpProfile;
use TencentCloud\Sms\V20210111\Models\SendSmsRequest;
use TencentCloud\Sms\V20210111\SmsClient;

class Qcloud
{
    public function handler($to,array $data)
    {
        go(function()use ($to,$data){
            // 实例化一个认证对象，入参需要传入腾讯云账户secretId，secretKey,此处还需注意密钥对的保密
            // 密钥可前往https://console.cloud.tencent.com/cam/capi网站进行获取
            $cred = new Credential((string)get_options('sms_qcloud_secret_id'), (string)get_options('sms_qcloud_secret_key'));
            // 实例化一个http选项，可选的，没有特殊需求可以跳过
            $httpProfile = new HttpProfile();
            $httpProfile->setEndpoint("sms.tencentcloudapi.com");

            // 实例化一个client选项，可选的，没有特殊需求可以跳过
            $clientProfile = new ClientProfile();
            $clientProfile->setHttpProfile($httpProfile);
            // 实例化要请求产品的client对象,clientProfile是可选的
            $client = new SmsClient($cred, "ap-guangzhou", $clientProfile);

            // 实例化一个请求对象,每个接口都会对应一个request对象
            $req = new SendSmsRequest();

            $params = array(
                "PhoneNumberSet" => $to,
                "SmsSdkAppId" => (string)get_options('sms_qcloud_sdk_app_id'),
                "SignName" => (string)get_options('sms_qcloud_sign_name'),
                "TemplateId" => (string)get_options('sms_qcloud_template'),
                "TemplateParamSet" => $data
            );
            $req->fromJsonString(json_encode($params));

            // 返回的resp是一个SendSmsResponse的实例，与请求对象对应
            // 输出json格式的字符串回包
            admin_log()->insert('sms','Qcloud','发信结果',[
                'data' => $client->SendSms($req)->toJsonString()
            ]);
            return $client->SendSms($req)->toJsonString();
        });
    }
}