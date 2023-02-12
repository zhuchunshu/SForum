<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Event\SendMail;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersNotice;
use Hyperf\Utils\Str;
use Psr\Http\Message\ResponseInterface;

class UserNotice
{
    // 检查用户是否愿意接受通知
    public function check($user_id)
    {
        $user_noticed = (string) get_user_settings($user_id, 'noticed', '0');
        if (get_options('user_email_noticed_on') === 'true' && $user_noticed === '1') {
            return true;
        }
        if (get_options('user_email_noticed_on') === 'false' && $user_noticed !== '1') {
            return true;
        }
        return false;
    }

    /**
     * 发送通知.
     * @param mixed $user_id
     * @param mixed $title
     * @param mixed $content
     * @param null|mixed $action
     */
    public function send($user_id, $title, $content, $action = null,bool $sendMail=true): void
    {
        if (UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->exists()) {
            UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->take(1)->update([
                'status' => 'publish',
                'created_at' => date('Y-m-d H:i:s'),
            ]);
        } else {
            UsersNotice::query()->create([
                'user_id' => $user_id,
                'title' => $title,
                'content' => $content,
                'action' => $action,
                'status' => 'publish',
            ]);
            if($sendMail===true){
                $this->sendMail($user_id, $title, $action, $content);
            }
        }
    }

    /**
     * 给多个用户发送通知.
     * @param mixed $title
     * @param mixed $content
     * @param null|mixed $action
     */
    public function sends(array $user_ids, $title, $content, $action = null,bool $sendMail=true): void
    {
        foreach ($user_ids as $user_id) {
            if (UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->exists()) {
                UsersNotice::query()->where(['user_id' => $user_id, 'content' => $content])->take(1)->update([
                    'status' => 'publish',
                    'created_at' => date('Y-m-d H:i:s'),
                ]);
            } else {
                UsersNotice::query()->create([
                    'user_id' => $user_id,
                    'title' => $title,
                    'content' => $content,
                    'action' => $action,
                    'status' => 'publish',
                ]);
                if($sendMail===true){
                    $this->sendMail($user_id, $title, $action, $content);
                }
            }
        }
    }

    /**
     * 发送邮件通知.
     * @param mixed $user_id
     * @param mixed $title
     * @param mixed $action
     * @param null|mixed $content
     */
    private function sendMail($user_id, $title, $action = null, $content = null): void
    {
        // 获取收件人邮箱
        $email = User::query()->where('id', $user_id)->first()->email;
        // 执行发送
        $Subject = '【' . get_options('web_name') . '】' . $title;
        // 获取发信内容
        $Body = $this->get_mail_content($title, $content, $action);
        // 判断用户是否愿意接收邮件通知
        // 检查用户是否愿意接受通知
        if ($this->check($user_id)) {
            // 发送邮件
            Email()->async_send($email, $Subject, $Body);
            // 触发发送事件
            EventDispatcher()->dispatch(new SendMail($user_id, $title, $action));
        }
    }

    // 获取发信内容
    private function get_mail_content($title, $content = null, $action = null): string
    {
        if ($content instanceof ResponseInterface) {
            $content = $content->getBody()->getContents();
        }
        $content = strip_tags($content);
        if (! Str::is('http*', $action)) {
            $url = url($action);
        } else {
            $url = $action;
        }
        // 执行发送
        if ($content) {
            $content = Str::limit($content, 10000, '...');
            $Body = <<<HTML
<h3>{$title}</h3>
<hr>
{$content}
<hr>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
        } else {
            $Body = <<<HTML
<h3>{$title}</h3>
<p>链接: <a href="{$url}">{$url}</a></p>
HTML;
        }
        return $Body;
    }
}
