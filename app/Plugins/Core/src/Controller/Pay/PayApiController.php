<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\Pay;

use App\Plugins\Core\src\Lib\Pay\Service\AliPay;
use App\Plugins\Core\src\Lib\Pay\Service\SFPay;
use App\Plugins\Core\src\Lib\Pay\Service\WechatPay;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\RequestMapping;

#[Controller(prefix: '/api/pay')]
class PayApiController
{
    #[RequestMapping(path: 'wechat/notify')]
    public function wechat_notify(): bool | array | \Psr\Http\Message\ResponseInterface
    {
        //admin_log()->insert('Pay','wechat','回调结果',1);
        $result = (new WechatPay())->notify(request()->all());
        if ($result === true) {
            return (new WechatPay())->pay()->success();
        }
        return $result;
    }

    #[RequestMapping(path: 'alipay/notify')]
    public function alipay_notify(): bool | array | \Psr\Http\Message\ResponseInterface
    {
        $result = (new AliPay())->notify(request()->all());
        if ($result === true) {
            return (new AliPay())->pay()->success();
        }
        return $result;
    }

    #[RequestMapping(path: 'SFPay/notify')]
    public function SFPay_notify()
    {
        $result = (new SFPay())->notify(request()->all());
        if ($result === true) {
            return Json_Api(200, true, ['msg' => 'success']);
        }
        return $result;
    }
}
