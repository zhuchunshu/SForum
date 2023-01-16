<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Controller;

use App\Plugins\Core\src\Handler\FileUpload;
use App\Plugins\Core\src\Handler\UploadHandler;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserFans;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersCollection;
use App\Plugins\User\src\Models\UsersNotice;
use App\Plugins\User\src\Models\UsersSetting;
use App\Plugins\User\src\Models\UserUpload;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller]
#[RateLimit(create: 1, capacity: 3)]
class ApiController
{
    #[PostMapping(path: '/user/upload/image')]
    #[Middleware(LoginMiddleware::class)]
    public function up_image(UploadHandler $uploader)
    {
        if (! Authority()->check('upload_file')) {
            return Json_Api(419, false, ['msg' => '你所在的用户组无权上传图片']);
        }
        $file = request()->file('file');
        if ($file->getSize() > get_options('core_user_up_img_size', 2048)) {
            $result = $uploader->save($file, 'topic', auth()->id());
            if ($result['success'] === true) {
                return Json_Api(200, true, ['msg' => '上传成功!', 'url' => $result['path']]);
            }
            return Json_Api(403, false, ['msg' => '上传失败!']);
        }
        return Json_Api(403, false, ['msg' => '上传失败!']);
    }

    #[PostMapping(path: '/user/upload/file')]
    #[Middleware(LoginMiddleware::class)]
    public function up_file(FileUpload $uploader)
    {
        $data = [];
        if (! Authority()->check('upload_file')) {
            return Json_Api(419, false, ['msg' => '你所在的用户组无权上传文件']);
        }
        foreach (request()->file('file') as $key => $file) {
            if ($file->getSize() > get_options('core_user_up_file_size', 4096)) {
                $result = $uploader->save($file, 'topic', auth()->id());
                if ($result) {
                    $url = $result['path'];
                    $data['data']['succMap'][$url] = $url;
                } else {
                    (array) $data['data']['errFiles'][] = $key;
                }
            }
        }
        return $data;
    }


    #[PostMapping(path: '/api/user/@has_user_username/{username}')]
    public function has_user_username($username): array
    {
        $username = urldecode($username);
        if (User::query()->where('username', $username)->count()) {
            return Json_Api(200, true, ['msg' => '此用户存在']);
        }
        return Json_Api(404, false, ['msg' => '用户:' . $username . '不存在']);
    }

    #[PostMapping(path: '/api/user/get.user.avatar.url')]
    public function get_user_avatar_url()
    {
        $user_id = request()->input('user_id');
        if (! $user_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:user_id']);
        }
        if (! User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['此用户不存在']);
        }
        $data = User::query()->where('id', $user_id)->first();
        return Json_Api(200, true, ['msg' => super_avatar($data)]);
    }

    #[PostMapping(path: '/api/user/get.user.data')]
    #[RateLimit(create: 12, capacity: 10)]
    public function get_user_data()
    {
        $user_id = request()->input('user_id');
        if (! $user_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:user_id']);
        }
        if (! User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['此用户不存在']);
        }
        $data = User::query()
            ->where('id', $user_id)
            ->with('Class', 'options')
            ->first();
        $data['avatar'] = super_avatar($data);
        $data['group'] = '<a href="/users/group/' . $data->class_id . '.html">' . Core_Ui()->Html()->UserGroup($data->Class) . '</a>';
        return Json_Api(200, true, $data);
    }

    #[PostMapping(path: '/api/user/get.user.config')]
    public function UserConfig(): array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }

        // 通知小红点
        $notice_red = 0;
        foreach (Itf()->get('users_notices') as $value) {
            $count = \Opis\Closure\unserialize((string) $value['count']);
            if (@$count && is_callable($count) && call_user_func($count, auth()->id()) > 0) {
                $notice_red += call_user_func($count, auth()->id());
            }
        }

        $config = [
            'notice_red' => $notice_red,
        ];
        return Json_Api(200, true, $config);
    }

    // 已读通知

    #[PostMapping(path: '/api/user/notice.read')]
    public function notice_read(): array
    {
        $notice_id = request()->input('notice_id');
        if (! $notice_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足']);
        }
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        if (! UsersNotice::query()->where(['status' => 'publish', 'user_id' => auth()->id(), 'id' => $notice_id])->exists()) {
            return Json_Api(403, false, ['msg' => '通知不存在!']);
        }
        UsersNotice::query()->where(['status' => 'publish', 'user_id' => auth()->id(), 'id' => $notice_id])->update([
            'status' => 'read',
        ]);
        return Json_Api(200, true, ['msg' => '设置成功!']);
    }

    // 一键清空未读通知

    #[PostMapping(path: '/api/user/notice.allread')]
    public function notice_allread(): array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        if (! UsersNotice::query()->where(['status' => 'publish', 'user_id' => auth()->id()])->exists()) {
            return Json_Api(403, false, ['msg' => '没有未读通知!']);
        }
        UsersNotice::query()->where(['status' => 'publish', 'user_id' => auth()->id()])->update([
            'status' => 'read',
        ]);
        return Json_Api(200, true, ['msg' => '一键清空未读通知成功!']);
    }

    // 关注用户

    #[PostMapping(path: '/api/user/userfollow')]
    public function user_follow()
    {
        $user_id = request()->input('user_id');
        if (! $user_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:user_id']);
        }
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }

        if (! User::query()->where('id', $user_id)->exists()) {
            return Json_Api(419, false, ['msg' => '用户不存在!']);
        }
        // 禁止关注自己
        if ($user_id == auth()->id()) {
            return Json_Api(419, false, ['msg' => '不能关注自己']);
        }

        if (UserFans::query()->where(['user_id' => $user_id, 'fans_id' => auth()->id()])->exists()) {
            UserFans::query()->where(['user_id' => $user_id, 'fans_id' => auth()->id()])->delete();

            // 发送取关通知
            user_notice()->send(
                $user_id,
                auth()->data()->username . ' 取关了你!',
                view('User::notice.userfollow_d')
            );
            return Json_Api(201, true, ['msg' => '已取关!']);
        }
        UserFans::query()->create(['user_id' => $user_id, 'fans_id' => auth()->id()]);

        // 发送通知
        user_notice()->send(
            $user_id,
            auth()->data()->username . ' 关注了你!',
            view('User::notice.userfollow')
        );

        return Json_Api(200, true, ['msg' => '已关注']);
    }

    // 查询关注状态

    #[PostMapping(path: '/api/user/userfollow.data')]
    public function user_follow_data()
    {
        $user_id = request()->input('user_id');
        if (! $user_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:user_id']);
        }
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        if (UserFans::query()->where(['user_id' => $user_id, 'fans_id' => auth()->id()])->exists()) {
            return Json_Api(200, true, ['msg' => '取关']);
        }
        return Json_Api(403, true, ['msg' => '关注']);
    }

    #[PostMapping(path: '/api/user/remove.collection')]
    public function remove_collection(): array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        $collection_id = request()->input('collection_id');
        if (! $collection_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:collection_id']);
        }
        if (! UsersCollection::query()->where('id', $collection_id)->exists()) {
            return Json_Api(403, false, ['msg' => '收藏id不存在']);
        }
        UsersCollection::query()->where('id', $collection_id)->delete();
        return Json_Api(200, true, ['msg' => '已取消收藏!']);
    }

    #[PostMapping(path: '/api/User/Files/remove')]
    public function filesRemove(): array
    {
        if (! admin_auth()->check()) {
            return Json_Api(419, false, ['msg' => '无权限!']);
        }
        $id = request()->input('id');
        if (! UserUpload::query()->where('id', $id)->exists()) {
            return Json_Api(403, false, ['msg' => '删除失败! 文件不存在!']);
        }
        $data = UserUpload::query()->where('id', $id)->first();
        if (! unlink($data->path)) {
            return Json_Api(403, false, ['msg' => '删除失败! 删除文件失败']);
        }
        if (! UserUpload::query()->where('id', $id)->delete()) {
            return Json_Api(403, false, ['msg' => '删除失败! 从数据库中删除记录失败!']);
        }
        return Json_Api(200, true, ['msg' => '删除成功!']);
    }

    #[PostMapping(path: '/api/user/get.user.settings')]
    public function get_user_settings()
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        $result = [];
        foreach (UsersSetting::query()->where('user_id', auth()->id())->select('name', 'value')->get() as $value) {
            $result[$value->name] = $value->value;
        }
        return Json_Api(200, true, $result);
    }

    #[PostMapping(path: '/api/user/set.user.settings')]
    public function set_user_settings()
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '未登录!']);
        }
        if (! is_array(request()->input('data'))) {
            $data = de_stringify(request()->input('data'));
        } else {
            $data = request()->input('data');
        }

        if (! is_array($data)) {
            return Json_Api(403, false, ['msg' => '请提交正确的数据']);
        }

        foreach ($data as $key => $value) {
            if (UsersSetting::query()->where(['user_id' => auth()->id(), 'name' => $key])->exists()) {
                UsersSetting::query()->where(['user_id' => auth()->id(), 'name' => $key])->update(['value' => $value]);
            } else {
                UsersSetting::query()->create(['user_id' => auth()->id(), 'name' => $key, 'value' => $value]);
            }
        }
        user_settings_clear(auth()->id());
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }

    #[PostMapping(path: '/api/User/get.session.ip')]
    public function get_user_session_ip(): array
    {
        $user_id = request()->input('user_id');
        if (! $user_id) {
            return Json_Api(403, false, ['msg' => '用户id不能为空']);
        }
        if (! User::query()->where('id', $user_id)->exists()) {
            return Json_Api(403, false, ['msg' => '用户不存在']);
        }
        $sessions = UsersAuth::query()->orderByDesc('created_at')->where('user_id', $user_id)->get();
        foreach ($sessions as $data) {
            if ($data->user_ip) {
                return Json_Api(200, true, ['msg' => core_default(get_client_ip_data($data->user_ip)['pro'], '未知')]);
            }
        }
        return Json_Api(403, false, ['msg' => '未找到用户IP归属地信息']);
    }
}
