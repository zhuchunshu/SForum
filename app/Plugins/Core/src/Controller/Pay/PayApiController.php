<?php

namespace App\Plugins\Core\src\Controller\Pay;

use App\Plugins\Core\src\Lib\Pay\Service\AliPay;
use App\Plugins\Core\src\Lib\Pay\Service\WechatPay;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;
use Psr\Http\Message\ServerRequestInterface;

#[Controller(prefix:"/api/pay")]
class PayApiController
{
    #[RequestMapping(path:"wechat/notify")]
    public function wechat_notify(): bool|array|\Psr\Http\Message\ResponseInterface
    {
        //admin_log()->insert('Pay','wechat','回调结果',1);
        $result = (new WechatPay())->notify(request()->all());
        if ($result===true) {
            return  (new WechatPay())->pay()->success();
        }
        return $result;
    }

    #[RequestMapping(path:"alipay/notify")]
    public function alipay_notify(): bool|array|\Psr\Http\Message\ResponseInterface
    {
        $result = (new AliPay())->notify(request()->all());
        if ($result===true) {
            return  (new AliPay())->pay()->success();
        }
        return $result;
    }
}
