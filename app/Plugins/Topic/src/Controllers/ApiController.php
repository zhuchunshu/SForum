<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicLike;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Models\TopicUpdated;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;
use Hyperf\Utils\Arr;
use Swoole\Timer;

#[Controller(prefix: '/api/topic')]
#[RateLimit(create: 1, capacity: 3)]
class ApiController
{
    #[PostMapping(path: 'tags')]
    public function Tags(): array
    {
        $data = [
            ['text' => '请选择标签', 'value' => 0],
        ];
        foreach (TopicTag::query()->where('status', '=', null)->get() as $key => $value) {
            $class_name = UserClass::query()->where('id', auth()->data()->class_id)->first()->name;
            if (user_TopicTagQuanxianCheck($value, $class_name)) {
                $data = Arr::add($data, $key + 1, [
                    'text' => $value->name,
                    'value' => $value->id,
                    //                    "icons" => "&lt;span class=&quot;avatar avatar-xs&quot; style=&quot;background-image: url($value->icon)&quot;&gt;&lt;/span&gt;"
                ]);
            }
        }
        return $data;
    }

    #[PostMapping(path: 'keywords')]
    public function topic_keyword(): array
    {
        $data = TopicKeyword::query()->select('name', 'id')->get();
        $arr = [];
        foreach ($data as $key => $value) {
            $arr = Arr::add($arr, $key, ['value' => '.[' . $value->name . ']', 'html' => $value->name]);
        }
        return $arr;
    }

    #[PostMapping(path: 'with_topic.data')]
    public function get_WithTopic_Data()
    {
        $topic_id = (int) request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['请求的 帖子id不存在']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(404, false, ['ID为:' . $topic_id . '帖子不存在']);
        }

        $data = Topic::query()->where('id', $topic_id)->select('id', 'title', 'user_id', 'options', 'created_at')->with('user')->first();
        $user_avatar = super_avatar($data->user);
        $title = \Hyperf\Utils\Str::limit($data->title, 20);
        $username = $data->user->username;
        $summary = '作者：' . $username;
        return Json_Api(200, true, [
            'avatar' => $user_avatar,
            'title' => $title,
            'summary' => $summary,
            'username' => $username,
        ]);
    }

    #[PostMapping('like.topic')]
    #[RateLimit(create: 1, capacity: 3)]
    public function like_topic()
    {
        if (! auth()->check()) {
            return Json_Api(403, false, ['msg' => '未登录!']);
        }

        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数:topic_id 不存在!']);
        }
        if (! Topic::query()->where([['id', $topic_id, 'status' => 'publish']])->exists()) {
            return Json_Api(403, false, ['msg' => 'id为:' . $topic_id . '的帖子不存在']);
        }
        if (TopicLike::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id(), 'type' => 'like'])->exists()) {
            TopicLike::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id(), 'type' => 'like'])->delete();
            Topic::query()->where(['id' => $topic_id])->decrement('like');
            // 发送通知
            $topic_data = Topic::query()->where('id', $topic_id)->first();
            if ($topic_data->user_id != auth()->id()) {
                $title = auth()->data()->username . '对你的帖子取消了点赞';
                $content = view('Topic::Notice.nolike_topic', ['user_data' => auth()->data(), 'data' => $topic_data]);
                $action = '/' . $topic_data->id . '.html';
                user_notice()->send($topic_data->user_id, $title, $content, $action);
            }
            return Json_Api(201, true, ['msg' => '已取消点赞']);
        }
        TopicLike::query()->create([
            'topic_id' => $topic_id,
            'user_id' => auth()->id(),
        ]);
        Topic::query()->where(['id' => $topic_id])->increment('like');
        // 发送通知
        $topic_data = Topic::query()->where('id', $topic_id)->first();
        if ($topic_data->user_id != auth()->id()) {
            $title = auth()->data()->username . '赞了你的帖子';
            $content = view('Topic::Notice.like_topic', ['user_data' => auth()->data(), 'data' => $topic_data]);
            $action = '/' . $topic_data->id . '.html';
            user_notice()->send($topic_data->user_id, $title, $content, $action);
        }
        return Json_Api(200, true, ['msg' => '已赞!']);
    }

    /**
     * 获取帖子信息.
     * @return array
     */
    #[PostMapping(path: 'topic.data')]
    public function topic_data()
    {
        if (! auth()->check()) {
            return Json_Api(403, false, ['msg' => '未登录!']);
        }
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => 'id为:' . $topic_id . '的帖子不存在']);
        }
        $data = Topic::query()->where('id', $topic_id)
            ->with('user', 'tag')
            ->first();
        $data['options'] = [];
        return Json_Api(200, true, $data);
    }

    // 设置精华

    #[PostMapping(path: 'set.topic.essence')]
    public function set_topic_essence(): array
    {
        $topic_id = request()->input('topic_id');
        $zhishu = request()->input('zhishu');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (empty($zhishu) && $zhishu != 0) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:zhishu']);
        }
        if (! auth()->check() || ! Authority()->check('topic_options')) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => '被操作的帖子不存在']);
        }
        if (! is_numeric($zhishu)) {
            return Json_Api(403, false, ['msg' => '精华指数必须为数字']);
        }
        if ($zhishu < 0 || $zhishu > 999) {
            return Json_Api(403, false, ['msg' => '精华指数 必须大于或等于0 并且小于或等于999']);
        }
        if ($zhishu === 0) {
            Topic::query()->where('id', $topic_id)->update([
                'essence' => null,
            ]);
        } else {
            Topic::query()->where('id', $topic_id)->update([
                'essence' => $zhishu,
            ]);
        }
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }

    // 设置置顶

    #[RateLimit(create: 1, capacity: 3)]
    #[PostMapping(path: 'set.topic.topping')]
    public function set_topic_topping(): array
    {
        $topic_id = request()->input('topic_id');
        $zhishu = request()->input('zhishu');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if ($zhishu != 0 && empty($zhishu)) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:zhishu']);
        }
        if (! auth()->check() || ! Authority()->check('topic_options')) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => '被操作的帖子不存在']);
        }
        if (! is_numeric($zhishu)) {
            return Json_Api(403, false, ['msg' => '置顶指数必须为数字']);
        }
        if ($zhishu < 0 || $zhishu > 999) {
            return Json_Api(403, false, ['msg' => '置顶指数 必须大于或等于0 并且小于或等于999']);
        }
        if ($zhishu === 0) {
            Topic::query()->where('id', $topic_id)->update([
                'topping' => null,
            ]);
        } else {
            Topic::query()->where('id', $topic_id)->update([
                'topping' => $zhishu,
            ]);
        }
        return Json_Api(200, true, ['msg' => '置顶成功!']);
    }

    // 删除帖子

    #[RateLimit(create: 1, capacity: 3)]
    #[PostMapping(path: 'set.topic.delete')]
    public function set_topic_delete(): array
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => '被删除的帖子不存在']);
        }
        $data = Topic::query()->where('id', $topic_id)->first();
        $quanxian = false;
        if (Authority()->check('admin_topic_delete') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            $quanxian = true;
        } elseif (Authority()->check('topic_delete') && auth()->id() === $data->user->id) {
            $quanxian = true;
        }

        if (! auth()->check() || $quanxian === false) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        Topic::query()->where('id', $topic_id)->update([
            'status' => 'delete',
        ]);
        return Json_Api(200, true, ['msg' => '已删除!']);
    }

    #[PostMapping(path: 'star.topic')]
    #[RateLimit(create: 1, capacity: 3)]
    public function star_topic(): array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => '要收藏的帖子不存在']);
        }
        if (UsersCollection::query()->where(['type' => 'topic', 'type_id' => $topic_id, 'user_id' => auth()->id()])->exists()) {
            UsersCollection::query()->where(['type' => 'topic', 'type_id' => $topic_id, 'user_id' => auth()->id()])->delete();
            return Json_Api(200, true, ['msg' => '取消收藏成功!']);
        }
        UsersCollection::query()->create([
            'user_id' => auth()->id(),
            'type' => 'topic',
            'type_id' => $topic_id,
        ]);
        return Json_Api(200, true, ['msg' => '已收藏']);
    }

    #[PostMapping(path: 'get.user')]
    #[RateLimit(create: 1, capacity: 3)]
    public function get_user(): array
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        $data = Topic::query()->where([
            'status' => 'publish',
            'id' => $topic_id,
        ])->with('user')->first();
        if (get_options('topic_author_ip', '开启') === '开启' && $data->post->user_ip) {
            $data['user']['city'] = __('app.IP attribution', ['province' => get_client_ip_data($data->post->user_ip)['pro']]);
        }
        return Json_Api(200, true, $data['user']);
    }

    // 获取修订者IP

    #[PostMapping(path: 'get.updated.user.ip')]
    #[RateLimit(create: 1, capacity: 3)]
    public function get_user_ip()
    {
        if (! request()->input('updateds')) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:updateds']);
        }
        if (! is_array(request()->input('updateds'))) {
            return Json_Api(403, false, ['msg' => '请求数据格式有误']);
        }
        $updateds = request()->input('updateds');
        $data = [];
        foreach ($updateds as $updated_id) {
            $updated = TopicUpdated::query()->where(['id' => $updated_id])->first();
            if ($updated->user_ip) {
                $data[] = [
                    'updated_id' => $updated->id,
                    'text' => 'IP归属地:' . core_default(get_client_ip_data($updated->user_ip)['pro'], '未知'),
                ];
            }
        }
        return Json_Api(200, true, $data);
    }
}
