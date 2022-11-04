<?php

namespace App\Plugins\Core\src\Lib\Sms\Service;

/**
 * 短信宝
 */
class SmsBao
{
    public function handler($to, array $data)
    {
        go(function () use ($to, $data) {
            $statusStr = array(
                "0" => "短信发送成功",
                "-1" => "参数不全",
                "-2" => "服务器空间不支持,请确认支持curl或者fsocket，联系您的空间商解决或者更换空间！",
                "30" => "密码错误",
                "40" => "账号不存在",
                "41" => "余额不足",
                "42" => "帐户已过期",
                "43" => "IP地址限制",
                "50" => "内容含有敏感词"
            );
            $content = '【' . get_options('sms_smsbao_name') . '】你的注册验证码为:' . $data[0] . '，' . $data[1] . '分钟内有效，如非本人操作，请忽略本短信';
            $http = http('raw')
                ->get('http://api.smsbao.com/sms?u=' .
                    get_options('sms_smsbao_user') .
                    '&p=' . md5(get_options('sms_smsbao_pass'))
                    . "&m=" . $to . '&c=' . $content)->getBody();
            $result = $http->getContents();
            admin_log()->insert('sms','SmsBao(短信宝)',$statusStr[$result],[
                'http' => $http,
                'result' => $result
            ]);
            return $statusStr[$result];

        });
    }
}