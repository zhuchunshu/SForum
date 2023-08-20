<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicLike;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Models\TopicUpdated;
use App\Plugins\User\src\Models\UserClass;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\DbConnection\Db;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;
use Hyperf\Collection\Arr;

#[Controller(prefix: '/api/topic')]
#[RateLimit(create: 1, capacity: 3)]
class ApiController
{
    #[PostMapping('tags')]
    public function Tags(): array
    {
        $data = [['text' => '请选择标签', 'value' => 0]];
        foreach (TopicTag::query()->where('status', '=', null)->get() as $key => $value) {
            $class_name = UserClass::query()->where('id', auth()->data()->class_id)->first()->name;
            if (user_TopicTagQuanxianCheck($value, $class_name)) {
                $data = Arr::add($data, $key + 1, ['text' => $value->name, 'value' => $value->id]);
            }
        }
        return $data;
    }

    #[PostMapping('keywords')]
    public function topic_keyword(): array
    {
        $data = TopicKeyword::query()->select('name', 'id')->get();
        $arr = [];
        foreach ($data as $key => $value) {
            $arr = Arr::add($arr, $key, ['value' => '.[' . $value->name . ']', 'html' => $value->name]);
        }
        return $arr;
    }

    #[PostMapping('with_topic.data')]
    #[RateLimit(create: 3, capacity: 6)]
    public function get_with_topic_data()
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
        $title = \Hyperf\Stringable\Str::limit($data->title, 20);
        $username = $data->user->username;
        $summary = '作者：' . $username;
        return Json_Api(200, true, ['avatar' => $user_avatar, 'title' => $title, 'summary' => $summary, 'username' => $username]);
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
        if (! Topic::query()->where([['id', $topic_id]])->exists()) {
            return Json_Api(403, false, ['msg' => 'id为:' . $topic_id . '的帖子不存在']);
        }
        if (TopicLike::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id(), 'type' => 'like'])->exists()) {
            TopicLike::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id(), 'type' => 'like'])->delete();
            // 发送通知
            $topic_data = Topic::find($topic_id);
            if ($topic_data->user_id != auth()->id()) {
                $title = auth()->data()->username . '对你的帖子取消了点赞';
                $content = view('Topic::Notice.nolike_topic', ['user_data' => auth()->data(), 'data' => $topic_data]);
                $action = '/' . $topic_data->id . '.html';
                user_notice()->send($topic_data->user_id, $title, $content, $action);
            }
            return Json_Api(201, true, ['msg' => '已取消点赞']);
        }
        TopicLike::query()->create(['topic_id' => $topic_id, 'user_id' => auth()->id()]);
        // 发送通知
        $topic_data = Topic::find($topic_id);
        if ($topic_data->user_id != auth()->id()) {
            $title = auth()->data()->username . '赞了你的帖子';
            $content = view('Topic::Notice.like_topic', ['data' => $topic_data]);
            $action = '/' . $topic_data->id . '.html';
            user_notice()->send($topic_data->user_id, $title, $content, $action);
        }
        if (get_options('topic_like_sort', 'false') === 'true') {
            Topic::where('id', $topic_id)->update(['last_time' => time()]);
        }
        return Json_Api(200, true, ['msg' => '已赞!']);
    }

    /**
     * 获取帖子信息.
     * @return array
     */
    #[PostMapping('topic.data')]
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
        $data = Topic::query()->where('id', $topic_id)->with('user', 'tag')->first();
        $data['options'] = [];
        return Json_Api(200, true, $data);
    }

    // 设置精华

    #[PostMapping('set.topic.essence')]
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
            Topic::query()->where('id', $topic_id)->update(['essence' => null]);
        } else {
            Topic::query()->where('id', $topic_id)->update(['essence' => $zhishu]);
        }
        return Json_Api(200, true, ['msg' => '更新成功!']);
    }

    // 设置置顶

    #[RateLimit(create: 1, capacity: 3)]
    #[PostMapping('set.topic.topping')]
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
            Topic::query()->where('id', $topic_id)->update(['topping' => null]);
        } else {
            Topic::query()->where('id', $topic_id)->update(['topping' => $zhishu]);
        }
        return Json_Api(200, true, ['msg' => '置顶成功!']);
    }

    // 锁帖

    #[RateLimit(create: 1, capacity: 3)]
    #[PostMapping('set.topic.lock')]
    public function set_topic_lock(): array
    {
        $topic_id = request()->input('topic_id');
        // 判断帖子是否存在
        if (! $topic_id || ! Topic::query()->where('id', $topic_id)->exists()) {
            return Json_Api(403, false, ['msg' => '帖子不存在']);
        }
        // 帖子信息
        $topic = Topic::find($topic_id);
        // 判断是否有权限
        if (! Authority()->check('topic_lock') && ! \App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $topic->tag_id)->where('user_id', auth()->id())->exists()) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        $tip = $topic->status === 'lock' ? '解除锁定' : '锁定';
        // 执行锁帖
        $update = $topic->status === 'lock' ? 'publish' : 'lock';
        Db::table('topic')->where('id', $topic_id)->update(['status' => $update]);
        // 发送通知
        if ($topic->user_id != auth()->id()) {
            user_notice()->send($topic->user_id, '你有一条帖子已被' . $tip, '帖子《<a href="/' . $topic_id . '.html" >' . $topic->title . '</a>》已被管理员：【<a href="/users/' . auth()->id() . '.html">' . auth()->data()->username . '</a>】' . $tip, '/' . $topic_id . '.html');
        }
        return Json_Api(200, true, ['msg' => $tip . '成功!']);
    }

    // 删除帖子

    #[RateLimit(create: 1, capacity: 3)]
    #[PostMapping('set.topic.delete')]
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
        } elseif (Authority()->check('topic_delete') && auth()->id() === (int) $data->user->id) {
            $quanxian = true;
        } elseif (\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->tag_id)->where('user_id', auth()->id())->exists()) {
            $quanxian = true;
        }
        if (! auth()->check() || $quanxian === false) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        // 删除评论
        go(function () use ($topic_id) {
            TopicComment::query()->where('topic_id', $topic_id)->delete();
        });
        // 删除帖子
        go(function () use ($topic_id) {
            Post::where('topic_id', $topic_id)->delete();
            Topic::destroy($topic_id);
        });
        // 发送通知
        if ($data->user_id != auth()->id()) {
            user_notice()->send($data->user_id, '你有一条帖子已被删除', '帖子《<a href="/' . $topic_id . '.html" >' . $data->title . '</a>》已被管理员：【<a href="/users/' . auth()->id() . '.html">' . auth()->data()->username . '</a>】删除', '/' . $topic_id . '.html', true, 'system');
        }
        return Json_Api(200, true, ['msg' => '已删除!']);
    }

    #[PostMapping('star.topic')]
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
        UsersCollection::query()->create(['user_id' => auth()->id(), 'type' => 'topic', 'type_id' => $topic_id]);
        return Json_Api(200, true, ['msg' => '已收藏']);
    }

    #[PostMapping('get.user')]
    #[RateLimit(create: 1, capacity: 3)]
    public function get_user(): array
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        $data = Topic::with('user', 'post')->find($topic_id);
        if (get_options('topic_author_ip', '开启') === '开启' && $data->post->user_ip) {
            $data['user']['city'] = __('app.IP attribution', ['province' => get_client_ip_data($data->post->user_ip)['pro']]);
        }
        return Json_Api(200, true, $data['user']);
    }

    // 获取修订者IP

    #[PostMapping('get.updated_user')]
    #[RateLimit(create: 1, capacity: 3)]
    public function get_user_ip()
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (! Topic::where(['id' => $topic_id])->exists()) {
            return admin_abort(['msg' => '帖子不存在', 'result' => []]);
        }
        $all = [];
        $topic_updated_author_ip = get_options('topic_updated_author_ip', '开启');
        foreach (TopicUpdated::where('topic_id', $topic_id)->orderByDesc('id')->get() as $item) {
            if ($topic_updated_author_ip === '开启') {
                $all[] = ['date' => $item->created_at, 'formatdate' => format_date($item->created_at), 'username' => $item->user->username, 'avatar' => avatar($item->user), 'ip' => get_client_ip_data($item->user_ip)['pro']];
            } else {
                $all[] = ['date' => $item->created_at, 'formatdate' => format_date($item->created_at), 'username' => $item->user->username, 'avatar' => avatar($item->user), 'ip' => '未知'];
            }
        }
        return json_api(200, true, ['msg' => '获取成功', 'result' => $all]);
    }

    // 获取帖子上下页信息

    #[PostMapping('get.topic.include.ifpage')]
    #[RateLimit(create: 1, capacity: 1)]
    public function get_topic_include_ifpage()
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:topic_id']);
        }
        if (! Topic::where(['id' => $topic_id])->exists()) {
            return admin_abort(['msg' => '帖子不存在', 'result' => []]);
        }
        $shang = Topic::where('id', '<', $topic_id)
            ->orderBy('id', 'desc')
            ->first(['title', 'id']);

        if ($shang) {
            $shang['url'] = "/{$shang['id']}.html";
            $shang['title'] = \Hyperf\Stringable\Str::limit($shang['title'] ?: ' ', 20, '...');
        }

        $xia = Topic::where('id', '>', $topic_id)
            ->orderBy('id', 'asc')
            ->first(['title', 'id']);

        if ($xia) {
            $xia['url'] = "/{$xia['id']}.html";
            $xia['title'] = \Hyperf\Stringable\Str::limit($xia['title'] ?: ' ', 20, '...');
        }

        $result = ['shang' => $shang, 'xia' => $xia];

        return json_api(200, true, ['msg' => '获取成功', 'result' => $result]);
    }
}
