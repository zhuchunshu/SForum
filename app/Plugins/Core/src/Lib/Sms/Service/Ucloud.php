<?php

namespace App\Plugins\Core\src\Lib\Sms\Service;

use UCloud\Core\Exception\UCloudException;
use UCloud\USMS\Apis\SendUSMSMessageRequest;
use UCloud\USMS\USMSClient;

class Ucloud
{
    public function handler($to,array $data){
        go(function() use ($to,$data){
            // Build client
            $client = new USMSClient([
                "publicKey" => get_options('sms_ucloud_publicKey'),
                "privateKey" => get_options('sms_ucloud_privateKey'),
                "projectId" => get_options('sms_ucloud_projectId'),
            ]);

            // Describe Image
            try {
                $req = new SendUSMSMessageRequest();
                $req->setPhoneNumbers($to);
                $req->setSigContent(get_options('sms_ucloud_sign_name'));
                $req->setTemplateId(get_options('sms_ucloud_template'));
                $req->setTemplateParams($data);
                $resp = $client->sendUSMSMessage($req);
            } catch (UCloudException $e) {
                throw $e;
            }
            return $resp->toArray();
        });
    }
}