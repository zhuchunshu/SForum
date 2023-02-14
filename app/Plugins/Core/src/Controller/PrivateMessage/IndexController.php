<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\PrivateMessage;

use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersPm;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;

#[Controller(prefix: '/users/pm')]
#[Middleware(LoginMiddleware::class)]
class IndexController
{
    #[GetMapping(path: '{user_id}')]
    public function index($user_id)
    {
        if (! Authority()->check('user_private_chat')) {
            return admin_abort('你所在的用户组无私聊权限');
        }
        if (! User::query()->where('id', $user_id)->exists()) {
            return redirect()->back()->with('danger', '用户不存在')->go();
        }
        $user = User::query()->find($user_id);
        if (get_user_settings($user_id, 'user_message_switch', '开启') !== '开启') {
            return redirect()->back()->with('info', '此用户未开启私信功能')->go();
        }
        if ((int) $user_id === auth()->id()) {
            return redirect()->back()->with('info', '不能私信自己')->go();
        }
        $messagesCount = UsersPm::query()->where([['from_id', auth()->id()], ['to_id', $user_id]])->Orwhere([['to_id', auth()->id()], ['from_id', $user_id]])->count();
        $messages = UsersPm::query()->where([['from_id', auth()->id()], ['to_id', $user_id]])->Orwhere([['to_id', auth()->id()], ['from_id', $user_id]])->with('from_user', 'to_user')->get();

        $contacts = [];
        foreach (UsersPm::query()->where(['from_id' => auth()->id()])->orWhere('to_id', auth()->id())->orderBy('created_at', 'desc')->get() as $pms) {
            if ((int) $pms->from_id !== auth()->id()) {
                $contacts[] = $pms->from_user;
            } elseif ((int) $pms->to_id !== auth()->id()) {
                $contacts[] = $pms->to_user;
            }
        }
        $contacts = array_unique($contacts);
        foreach ($contacts as $key => $value) {
            $count = \App\Plugins\User\src\Models\UsersPm::query()->where(['from_id' => $value->id, 'to_id' => auth()->id(), 'read' => false])->count();
            $contacts[$key]['msgCount'] = $count;
        }
        $msgExists = UsersPm::query()->where([['from_id', auth()->id()], ['to_id', $user_id]])->Orwhere([['to_id', auth()->id()], ['from_id', $user_id]])->exists();
        return view('User::pm.index', ['user' => $user, 'messagesCount' => $messagesCount, 'messages' => $messages, 'contacts' => $contacts, 'msgExists' => $msgExists]);
    }
}
