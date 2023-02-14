<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller\Api;

use App\Plugins\Topic\src\ContentParse;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersPm;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\Utils\Str;

/**
 * 私信
 */
#[Middleware(LoginMiddleware::class)]
#[Controller(prefix: '/api/user/pm')]
class PmController
{
    /**
     * 发送消息.
     */
    #[PostMapping(path: 'send_msg')]
    public function send_msg()
    {
        if (! Authority()->check('user_private_chat')) {
            return admin_abort('你所在的用户组无私聊权限');
        }
        $user_id = request()->input('user_id');
        $msg = request()->input('content');
        if (! $user_id || ! User::query()->where('id', $user_id)->exists()) {
            return json_api(403, false, ['msg' => '私信用户不存在']);
        }
        if (Str::length($msg) > get_options('pm_msg_maxlength', 300)) {
            return json_api(403, false, ['msg' => '内容长度超出闲置']);
        }
        if (get_user_settings($user_id, 'user_message_switch', '开启') !== '开启') {
            return json_api(403, false, ['msg' => '用户未开启私信功能']);
        }
        UsersPm::query()->create([
            'from_id' => auth()->id(),
            'to_id' => $user_id,
            'message' => htmlspecialchars($msg),
        ]);
        return json_api(200, true, ['msg' => '发送成功!']);
    }

    #[PostMapping(path: 'get_msg')]
    public function get_msg()
    {
        if (! Authority()->check('user_private_chat')) {
            return admin_abort('你所在的用户组无私聊权限');
        }
        $user_id = request()->input('user_id');
        if (! $user_id || ! User::query()->where('id', $user_id)->exists()) {
            return json_api(403, false, ['msg' => '私信用户不存在']);
        }
        if (get_user_settings($user_id, 'user_message_switch', '开启') !== '开启') {
            return json_api(403, false, ['msg' => '用户未开启私信功能']);
        }
        $auth_id = auth()->id();
        go(function () use ($user_id, $auth_id) {
            UsersPm::query()->where('to_id', $auth_id)->where('from_id', $user_id)->update(['read' => true]);
            UsersPm::query()->where(['from_id' => $user_id, 'to_id' => $auth_id])->update(['read' => true]);
        });
        $message = [];

        $msg = UsersPm::query()->where([['from_id', auth()->id()], ['to_id', $user_id]])->Orwhere([['to_id', auth()->id()], ['from_id', $user_id]])->get();
        foreach ($msg as $data) {
            unset($data['updated_at'],$data['read']);
            $data['message']=ContentParse()->parse($data['message']);
            $message[$data['id']] = $data;
        }
        return Json_Api(200, true, ['msg' => '消息获取成功', 'message' => $message]);
    }
}
