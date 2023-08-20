<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Controller\User;

use App\Plugins\Core\src\Handler\AvatarUpload;
use App\Plugins\Core\src\Handler\UploadHandler;
use App\Plugins\Core\src\Request\User\Mydata\AvatarRequest;
use App\Plugins\Core\src\Request\User\Mydata\JibenRequest;
use App\Plugins\Core\src\Request\User\Mydata\OptionsRequest;
use App\Plugins\User\src\Event\Task\System\SetAvatar;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserRepwd;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\HttpServer\Annotation\RequestMapping;
use HyperfExt\Hashing\Hash;
use Hyperf\Stringable\Str;
use Psr\Http\Message\ResponseInterface;
#[Controller]
#[Middleware(LoginMiddleware::class)]
class UpdateController
{
    // 个人设置
    #[GetMapping('/user/setting')]
    public function user_setting()
    {
        $data = User::query()->where('id', auth()->id())->with('Class', 'Options')->first();
        $itf = Itf()->get('userSetting');
        return view('App::user.setting', ['data' => $data, 'itf' => $itf]);
    }
    #[PostMapping('/user/data')]
    public function my_data()
    {
        return auth()->data();
    }
    // 更新个人信息
    #[PostMapping('/user/myUpdate')]
    public function myUpdate(JibenRequest $request)
    {
        if (!$request->input('old_pwd') || !$request->input('new_pwd')) {
            return redirect()->back()->with('info', '无修改')->go();
        }
        $old_pwd = $request->input('old_pwd');
        $new_pwd = $request->input('new_pwd');
        if (!Hash::check($old_pwd, auth()->data()->password)) {
            return redirect()->back()->with('danger', '旧密码错误')->go();
        }
        $pwd = Hash::make($new_pwd);
        $data = UserRepwd::query()->create(['user_id' => auth()->data()->id, 'pwd' => $pwd, 'hash' => Str::random()]);
        $user = auth()->data();
        go(static function () use($data, $user) {
            $url = url('/user/myUpdate/ConfirmPassword/' . $data->id . '/' . $data->hash);
            $Subject = '【' . get_options('web_name') . '】修改密码确认';
            $Body = <<<HTML
你好 {$user->username},<br>
你在本网站修改了用户密码,安全起见点击以下链接确认修改:<br>
<a href="{$url}">{$url}</a>
HTML;
            Email()->send($user->email, $Subject, $Body);
        });
        return redirect()->url('/user/setting')->with('success', '修改密码邮件已发送至你的邮箱')->go();
    }
    /**
     * 处理修改密码
     * @param $id
     * @param $hash
     * @return ResponseInterface
     */
    #[GetMapping('/user/myUpdate/ConfirmPassword/{id}/{hash}')]
    public function myUpdate_ConfirmPassword($id, $hash) : ResponseInterface
    {
        if (!UserRepwd::query()->where(['user_id' => auth()->data()->id, 'id' => $id, 'hash' => $hash])->count()) {
            return admin_abort(['msg' => '鉴权失败,无法修改']);
        }
        $data = UserRepwd::query()->where(['user_id' => auth()->data()->id, 'id' => $id, 'hash' => $hash])->first();
        User::query()->where(['id' => $data->user_id])->update(['password' => $data->pwd]);
        auth()->logout();
        UserRepwd::query()->where(['id' => $id, 'hash' => $hash])->delete();
        return redirect()->url('/')->with('success', '密码修改成功,请重新登录!')->go();
    }
    /**
     * 上传头像.
     */
    #[PostMapping('/user/myUpdate/avatar')]
    public function update_avatar(AvatarRequest $request, AvatarUpload $upload)
    {
        $data = $upload->save($request->file('avatar'), auth()->id(), \Hyperf\Stringable\Str::random(), 200);
        $path = $data['path'];
        User::query()->where('id', auth()->id())->update(['avatar' => $path]);
        // 头像上传成功
        // 触发头像上传成功事件
        EventDispatcher()->dispatch(new SetAvatar(auth()->id()));
        return redirect()->url('/user/setting')->with('success', '头像修改成功')->go();
    }
    #[PostMapping('/user/myUpdate/other')]
    public function update_action()
    {
        $action = request()->input('action');
        if (!$action) {
            return redirect()->url('/user/setting')->with('danger', 'action 为空!')->go();
        }
        // 删除头像
        if ($action === 'removeAvatar') {
            User::query()->where('id', auth()->id())->update(['avatar' => null]);
            return redirect()->url('/user/setting')->with('success', '头像删除成功!')->go();
        }
        return redirect()->url('/user/setting')->with('danger', '当前 action 处理方法不存在')->go();
    }
    #[PostMapping('/user/myUpdate/options')]
    public function update_options(OptionsRequest $request)
    {
        $data = $request->validated();
        UsersOption::query()->where(['id' => auth()->data()->options_id])->update($data);
        return redirect()->url('/user/setting')->with('success', '更新成功!')->go();
    }
    #[PostMapping('/user/myUpdate/noticed')]
    public function update_noticed()
    {
        $user_id = auth()->id();
        if (get_user_settings($user_id, 'noticed', '0') === '1') {
            set_user_settings($user_id, ['noticed' => '0']);
        } else {
            set_user_settings($user_id, ['noticed' => '1']);
        }
        return redirect()->url('/user/setting')->with('success', '更新成功!')->go();
    }
    #[PostMapping('/user/setbackgroundImg')]
    public function setBackgroundImg(UploadHandler $uploader)
    {
        if (!request()->hasFile('backgroundImg')) {
            return redirect()->url('/user/setting?m=userSetting_3')->with('danger', '背景图片上传失败!')->go();
        }
        $file = $uploader->save(request()->file('backgroundImg'), 'backgroundImg/', auth()->id(), 1200);
        $path = $file['path'] ?: null;
        if ($path) {
            set_user_settings(auth()->id(), ['backgroundImg' => $path]);
            return redirect()->url('/user/setting?m=userSetting_3')->with('success', '背景图片修改成功')->go();
        }
        return redirect()->url('/user/setting?m=userSetting_3')->with('danger', '背景图片修改失败!')->go();
    }
}