<?php

namespace App\Plugins\User\src\Listener;
use App\Plugins\User\src\Event\SendNotice;
use App\Plugins\User\src\Models\User;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;

#[Listener]
class SendNoticeListener implements ListenerInterface
{
    public function listen(): array
    {
        // 返回一个该监听器要监听的事件数组，可以同时监听多个事件
        return [
            SendNotice::class,
        ];
    }

    /**
     * @param SendNotice $event
     */
    public function process(object $event)
    {
        /**
         * 发送邮件通知
         */
        // 接受者邮箱
        $email = User::query()->where('id', $event->user_id)->first()->email;
        $mail = Email();

        // 判断用户是否愿意接收通知
        if(user_notice()->check("email",$event->user_id)===false){
            // 执行发送
            go(function() use ($event,$email,$mail){
                $mail->addAddress($email);
                $mail->Subject = "【".get_options("web_name")."】 你有一条新通知!";
                $mail->Body    = <<<HTML
<h3>标题: {$event->title}</h3>
<p>链接: {$event->action}</p>
HTML;
                $mail->send();
            });
        }


    }
}