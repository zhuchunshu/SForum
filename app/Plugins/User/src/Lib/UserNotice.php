<?php

namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Event\SendMail;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersNotice;
use App\Plugins\User\src\Models\UsersNoticed;
use Hyperf\Database\Schema\Schema;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Utils\Str;
use Psr\EventDispatcher\EventDispatcherInterface;

class UserNotice
{
    
    /**
     * @Inject
     * @var EventDispatcherInterface
     */
    private $eventDispatcher;
    
    /**
     * 返回值为true 则证明用户不愿意接受通知
     * @param string $type
     * @param int|string $user_id
     * @return bool
     */
    public function check(string $type, int|string $user_id): bool
    {
        if (!User::query()->where("id", $user_id)->exists()) {
            return false;
        }
        if (!UsersNoticed::query()->where("user_id", $user_id)->exists()) {
            return false;
        }
        if (!Schema::hasColumn("users_noticed", $type)) {
            return false;
        }
        if (UsersNoticed::query()->where(["user_id" => $user_id, $type => true])->exists()) {
            return true;
        }
        return false;
    }
    
    public function checked(string $type, int|string $user_id): string
    {
        if ($this->check($type, $user_id)) {
            return "checked";
        }
        return "";
    }
    
    public function update($user_id, $data): void
    {
        UsersNoticed::query()->where(["user_id" => $user_id])->delete();
        foreach ($data as $key => $value) {
            if ($value === 'on') {
                $value = true;
            } else {
                $value = false;
            }
            if (Schema::hasColumn("users_noticed", $key)) {
                UsersNoticed::query()->where(["user_id" => $user_id])->create([
                    $key => $value,
                    "user_id" => $user_id
                ]);
            }
        }
    }
    
    /**
     * 发送通知
     */
    public function send($user_id, $title, $content, $action = null, $contentLength =197): void
    {
        if (UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->exists()) {
            UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->take(1)->update([
                'status' => 'publish',
                'created_at' => date('Y-m-d H:i:s')
            ]);
        } else {
            UsersNotice::query()->create([
                'user_id' => $user_id,
                'title' => $title,
                'content' => $content,
                'action' => $action,
                'status' => 'publish'
            ]);
        }
        
        $this->sendMail($user_id, $title, $action, $content, $contentLength);
    }
    
    
    /**
     * 给多个用户发送通知
     */
    public function sends(array $user_ids, $title, $content, $action = null, $contentLength =197): void
    {
        foreach ($user_ids as $user_id) {
            if (UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->exists()) {
                UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->take(1)->update([
                    'status' => 'publish',
                    'created_at' => date('Y-m-d H:i:s')
                ]);
            } else {
                UsersNotice::query()->create([
                    'user_id' => $user_id,
                    'title' => $title,
                    'content' => $content,
                    'action' => $action,
                    'status' => 'publish'
                ]);
            }
            $this->sendMail($user_id, $title, $action, $content, $contentLength);
        }
    }
    
    private function sendMail($user_id, $title, $action, $content=null, $contentLength =197)
    {
        $email = User::query()->where('id', $user_id)->first()->email;
        $mail = Email();
        $url = url($action);
        // 判断用户是否愿意接收通知
        $_this = $this;
        go(function () use ($title, $mail, $url, $email, $_this, $user_id, $content, $contentLength) {
            //			 新版判断false , 如果check结果为true 证明用户不愿意接收邮件通知
            // 如果默认开启所有人邮件通知
            if (get_options('user_email_noticed_on', 'false') === "true") {
                if ($_this->check("email", $user_id) === false) {
                    // 执行发送
                    $mail->addAddress($email);
                    $mail->Subject = "【" . get_options("web_name") . "】 你有一条新通知!";
                    if ($content) {
                        $content = Str::limit($content, $contentLength);
                        $mail->Body = <<<HTML
<h3>标题: {$title}</h3>
<hr>
{$content}
<hr>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                    } else {
                        $mail->Body = <<<HTML
<h3>标题: {$title}</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                    }
                    $mail->send();
                    // 触发发送事件
                    EventDispatcher()->dispatch(new SendMail($user_id, $title, $url));
                }
            } elseif ($_this->check("email", $user_id) === true) {
                // 执行发送
                $mail->addAddress($email);
                $mail->Subject = "【" . get_options("web_name") . "】 你有一条新通知!";
                if ($content) {
                    $content = Str::limit($content, $contentLength);
                    $mail->Body = <<<HTML
<h3>标题: {$title}</h3>
<hr>
{$content}
<hr>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                } else {
                    $mail->Body = <<<HTML
<h3>标题: {$title}</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
                }
                $mail->send();
                // 触发发送事件
                EventDispatcher()->dispatch(new SendMail($user_id, $title, $url));
            }
        });
    }
}
