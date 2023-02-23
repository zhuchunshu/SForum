<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\SendService;

use AlibabaCloud\SDK\Dm\V20151123\Dm;
use AlibabaCloud\SDK\Dm\V20151123\Models\SingleSendMailRequest;
use AlibabaCloud\Tea\Exception\TeaError;
use AlibabaCloud\Tea\Utils\Utils\RuntimeOptions;
use App\Plugins\Mail\src\Annotation\SendService;
use App\Plugins\Mail\src\SendServiceInterface;
use Darabonba\OpenApi\Models\Config;

#[SendService]
class Aliyun implements SendServiceInterface
{
    public function get_service_name(): string
    {
        return '阿里云邮件推送';
    }

    public function view(): string
    {
        return 'Mail::config.aliyun';
    }

    public function handler(): string
    {
        return \App\Plugins\Mail\src\Handler\Aliyun::class;
    }

    public function send($email, $subject, $body)
    {
        // 工程代码泄露可能会导致AccessKey泄露，并威胁账号下所有资源的安全性。以下代码示例仅供参考，建议使用更安全的 STS 方式，更多鉴权访问方式请参见：https://help.aliyun.com/document_detail/311677.html
        $client = $this->createClient(get_options('MAIL_Ali_accessKeyId'), get_options('MAIL_Ali_accessKeySecret'));
        $replyToAddress = true;
        if (get_options('MAIL_Ali_replyToAddress', '开启') === '关闭') {
            $replyToAddress = false;
        }
        $singleSendMailRequest = new SingleSendMailRequest([
            'accountName' => get_options('MAIL_Ali_AccountName'),
            'replyToAddress' => $replyToAddress,
            'toAddress' => $email,
            'subject' => $subject,
            'htmlBody' => $body,
            'fromAlias' => get_options('MAIL_Ali_FromAlias'),
            'replyAddress' => get_options('MAIL_Ali_ReplyAddress'),
            'replyAddressAlias' => get_options('MAIL_Ali_ReplyAddressAlias'),
            'addressType' => (int) get_options('MAIL_Ali_AddressType') ?: 0,
        ]);
        $runtime = new RuntimeOptions([]);
        try {
            // 复制代码运行请自行打印 API 的返回值
            if((int)$client->singleSendMailWithOptions($singleSendMailRequest, $runtime)->statusCode===200){
                return true;
            }

        } catch (\Exception $error) {
            if (! ($error instanceof TeaError)) {
                $error = new TeaError([], $error->getMessage(), $error->getCode(), $error);
            }
            // 如有需要，请打印 error
            admin_log()->insert('mail', 'sendMail(Aliyun发信)', '发信失败!', [
                'error' => $error,
            ]);
            throw $error;
        }
        return false;
    }

    /**
     * 使用AK&SK初始化账号Client.
     * @param string $accessKeyId
     * @param string $accessKeySecret
     * @return Dm Client
     */
    private function createClient($accessKeyId, $accessKeySecret): Dm
    {
        $config = new Config([
            // 必填，您的 AccessKey ID
            'accessKeyId' => $accessKeyId,
            // 必填，您的 AccessKey Secret
            'accessKeySecret' => $accessKeySecret,
        ]);
        // 访问的域名
        $config->endpoint = 'dm.aliyuncs.com';
        return new Dm($config);
    }
}
