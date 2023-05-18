<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Comment\src\Model\TopicCommentLike;
use App\Plugins\Comment\src\Request\TopicReply;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\RateLimit\Annotation\RateLimit;

#[Controller(prefix: '/api/comment')]
#[RateLimit(create: 1, capacity: 3)]
class ApiController
{
    // 回复评论

    #[PostMapping(path: 'comment.topic.reply')]
    public function topic_reply_create(TopicReply $request): bool | array
    {
        // 鉴权
        if ($this->comment_reply_validation() !== true) {
            return $this->comment_reply_validation();
        }
        // 处理

        // 原内容
        $yhtml = $request->input('content');

        // 过滤xss
        $content = xss()->clean($request->input('content'));

        // 解析艾特
        $content = $this->topic_create_at($content);

        $comment_id = request()->input('comment_id');
        if (! TopicComment::query()->where('id', $comment_id)->exists()) {
            return json_api(404,false,'评论不存在');
        }
        $topic_id = TopicComment::query()->where('id', $comment_id)->first()->topic_id;
        $topic = Topic::query()->find($topic_id);
        if (@$topic->post->options->disable_comment) {
            return json_api(403,false,'此帖子关闭了评论功能');
        }
        $parent_id = TopicComment::query()->where('id', $comment_id)->first()->user_id;
        $post = Post::query()->create([
            'content' => $content,
            'user_agent' => get_user_agent(),
            'user_ip' => get_client_ip(),
            'user_id' => auth()->id(),
        ]);
        $data = TopicComment::query()->create([
            'post_id' => $post->id,
            'parent_url' => '/' . $topic_id . '.html/' . $comment_id . '?page=' . get_topic_comment_page((int) $comment_id),
            'topic_id' => $topic_id,
            'parent_id' => $comment_id,
            'user_id' => auth()->id(),
        ]);
        // 给posts表设置comment_id字段的值
        Post::query()->where('id', $post->id)->update(['comment_id' => $data->id]);
        $data = TopicComment::query()->where('id', $data->id)->first();
        Topic::query()->where('id', $data['topic_id'])->update(['last_time' => time()]);

        // 艾特被回复的人
        $this->at_user($data, $yhtml);

        //发布成功
        // 发送通知 - 帖子作者
        $topic_data = Topic::query()->where('id', $topic_id)->first();
        if ($topic_data->user_id != auth()->id() && $topic_data->user_id != $parent_id) {
            $title = auth()->data()->username . '评论了你发布的帖子!';
            $content = view('Comment::Notice.comment', ['comment' => $content, 'user_data' => auth()->data(), 'data' => $data]);
            $action = '/' . $topic_data->id . '.html';
            user_notice()->send($topic_data->user_id, $title, $content, $action);
        }
        // 发送通知 - 被回复的人
        if ($parent_id != auth()->id()) {
            $title = auth()->data()->username . '回复了你的评论!';
            $content = view('Comment::Notice.reply', ['comment' => $content, 'user_data' => auth()->data(), 'data' => $data]);
            $action = '/' . $topic_data->id . '.html';
            user_notice()->send($parent_id, $title, $content, $action);
        }

        // 设置下一此回复时间
        cache()->set('comment_create_time_' . auth()->id(), time() + get_options('comment_create_time', 60), get_options('comment_create_time', 60));

        return Json_Api(200, true, ['msg' => '回复成功!', 'url' => '/' . $data->topic_id . '.html/' . $data->id . '?page=' . get_topic_comment_page($data->id)]);
    }

    // 删除帖子评论

    #[PostMapping(path: 'comment.topic.delete')]
    public function topic_delete()
    {
        $comment_id = request()->input('comment_id');
        if (! auth()->check()) {
            return Json_Api(419, false, ['未登录']);
        }
        if (! $comment_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:comment_id']);
        }
        if (! TopicComment::query()->where('id', $comment_id)->exists()) {
            return Json_Api(403, false, ['id为' . $comment_id . '的评论不存在']);
        }
        $data = TopicComment::query()->find($comment_id);
        $quanxian = false;
        if (Authority()->check('admin_comment_remove') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            $quanxian = true;
        }
        if (Authority()->check('comment_remove') && auth()->id() === (int)$data->user->id) {
            $quanxian = true;
        }
        if(\App\Plugins\Topic\src\Models\Moderator::query()->where('tag_id', $data->topic->tag_id)->where('user_id',auth()->id())->exists()){
            $quanxian = true;
        }
        if ($quanxian === false) {
            return Json_Api(419, false, ['无权限!']);
        }
        TopicComment::query()->where('id', $comment_id)->delete();
        return Json_Api(200, true, ['已删除!']);
    }

    // 对帖子进行评论 -- 鉴权
    public function comment_reply_validation(): bool | array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg'=>'未登录']);
        }
        if (! Authority()->check('comment_create')) {
            return Json_Api(419, false, ['msg'=>'无评论权限']);
        }
        if (cache()->has('comment_create_time_' . auth()->id())) {
            $time = cache()->get('comment_create_time_' . auth()->id()) - time();
            return Json_Api(419, false, ['msg'=>'发表评论过于频繁,请 ' . $time . ' 秒后再试']);
        }
        return true;
    }

    #[PostMapping(path: 'get.topic.comment')]
    public function topic_comment_list()
    {
        $topic_id = request()->input('topic_id');
        if (! $topic_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:topic_id']);
        }
        if (! Topic::query()->where(['id' => $topic_id])->exists()) {
            return Json_Api(403, false, ['ID为:' . $topic_id . '的帖子不存在']);
        }
        if (! TopicComment::query()->where(['topic_id' => $topic_id])->count()) {
            return Json_Api(403, false, ['此帖子下无评论']);
        }
        $page = TopicComment::query()
            ->where(['topic_id' => $topic_id])
            ->with('topic', 'user')
            ->paginate((int) get_options('comment_page_count', 2));
        return Json_Api(200, true, $page);
    }

    #[PostMapping('like.topic.comment')]
    public function like_topic_comment(): array
    {
        if (! auth()->check()) {
            return Json_Api(403, false, ['msg' => '未登录!']);
        }

        $comment_id = request()->input('comment_id');
        if (! $comment_id) {
            return Json_Api(403, false, ['msg' => '请求参数:comment_id 不存在!']);
        }
        if (! TopicComment::query()->where('id', $comment_id)->exists()) {
            return Json_Api(403, false, ['msg' => 'id为:' . $comment_id . '的评论不存在']);
        }
        if (TopicCommentLike::query()->where(['comment_id' => $comment_id, 'user_id' => auth()->id()])->exists()) {
            TopicCommentLike::query()->where(['comment_id' => $comment_id, 'user_id' => auth()->id()])->delete();
            return Json_Api(201, true, ['msg' => '已取消点赞!']);
        }
        TopicCommentLike::query()->create([
            'comment_id' => $comment_id,
            'user_id' => auth()->id(),
        ]);
        return Json_Api(200, true, ['msg' => '已赞!']);
    }

    #[PostMapping(path: 'topic.comment.data')]
    public function topic_comment_data(): array
    {
        $comment_id = request()->input('comment_id');
        if (! $comment_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:comment_id']);
        }
        if (! TopicComment::query()->where([['id', $comment_id]])->exists()) {
            return Json_Api(403, false, ['id为:' . $comment_id . '的评论不存在']);
        }
        $data = TopicComment::query()
            ->where([['id', $comment_id]])
            ->first();
        return Json_Api(200, true, $data);
    }

    #[PostMapping('topic.caina.comment')]
    public function topic_caina_comment()
    {
        $comment_id = request()->input('comment_id');
        if (! $comment_id) {
            return Json_Api(403, false, ['请求参数不足,缺少:comment_id']);
        }
        if (! TopicComment::query()->where([['id', $comment_id]])->exists()) {
            return Json_Api(403, false, ['id为:' . $comment_id . '的评论不存在']);
        }
        $data = TopicComment::query()
            ->where([['id', $comment_id]])
            ->with('topic')
            ->first();
        $quanxian = false;
        if ($data->topic->user_id == auth()->id() && Authority()->check('comment_caina')) {
            $quanxian = true;
        }
        if (Authority()->check('admin_comment_caina')) {
            $quanxian = true;
        }
        if ($quanxian === false) {
            return Json_Api(419, false, ['无权限!']);
        }
        $caina = __('topic.comment.cancel') . ' ' . __('topic.comment.adoption');
        if ($data->optimal === null) {
            TopicComment::query()->where([['id', $comment_id]])->update([
                'optimal' => date('Y-m-d H:i:s'),
            ]);
            $caina = __('topic.comment.adoption');
        } else {
            TopicComment::query()->where([['id', $comment_id]])->update([
                'optimal' => null,
            ]);
        }
        $topic = Topic::query()->where('id', $data->topic_id)->first();
        $url = get_topic_comment_url($comment_id);
        user_notice()->send($data->user_id, auth()->data()->username . $caina . '了你的评论', '你发布在: <h2>' . $topic->title . '</h2> 的评论已被' . $caina, $url);
        return Json_Api(200, true, ['更新成功!']);
    }

    // 收藏评论

    #[PostMapping(path: 'star.comment')]
    #[RateLimit(create: 1, capacity: 3)]
    public function star_topic(): array
    {
        if (! auth()->check()) {
            return Json_Api(419, false, ['msg' => '权限不足!']);
        }
        $comment_id = request()->input('comment_id');
        if (! $comment_id) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:comment_id']);
        }
        if (! TopicComment::query()->where('id', $comment_id)->exists()) {
            return Json_Api(403, false, ['msg' => '要收藏的评论不存在']);
        }
        if (UsersCollection::query()->where(['type' => 'comment', 'type_id' => $comment_id, 'user_id' => auth()->id()])->exists()) {
            UsersCollection::query()->where(['type' => 'comment', 'type_id' => $comment_id, 'user_id' => auth()->id()])->delete();
            return Json_Api(200, true, ['msg' => '取消收藏成功!']);
        }
        UsersCollection::query()->create([
            'user_id' => auth()->id(),
            'type' => 'comment',
            'type_id' => $comment_id,
        ]);
        return Json_Api(200, true, ['msg' => '已收藏']);
    }

    // 获取评论者IP

    #[PostMapping(path: 'get.user.ip')]
    #[RateLimit(create: 1, capacity: 3)]
    public function get_user_ip()
    {
        if (! request()->input('comments')) {
            return Json_Api(403, false, ['msg' => '请求参数不足,缺少:comments']);
        }
        if (! is_array(request()->input('comments'))) {
            return Json_Api(403, false, ['msg' => '请求数据格式有误']);
        }
        $comments = request()->input('comments');
        $data = [];
        foreach ($comments as $comment_id) {
            $comment = TopicComment::query()->where(['id' => $comment_id, ])->first();
            if ($comment->post->user_ip) {
                $data[] = [
                    'comment_id' => $comment->id,
                    'text' => __('app.IP attribution', ['province' => get_client_ip_data($comment->post->user_ip)['pro']]),
                ];
            }
        }
        return Json_Api(200, true, $data);
    }

    private function at_user(\Hyperf\Database\Model\Model | \Hyperf\Database\Model\Builder $data, string $html): void
    {
        $at_user = get_all_at($html);
        foreach ($at_user as $username) {
            go(function () use ($username, $data) {
                if (User::query()->where('username', $username)->exists()) {
                    $user = User::query()->where('username', $username)->first();
                    if ((int) $user->id !== (int) $data->user_id) {
                        user_notice()->send(
                            $user->id,
                            '有人在评论中提到了你',
                            '有人在评论中提到了你',
                            get_topic_comment_url($data->id)
                        );
                    }
                }
            });
        }
    }

    // 解析创建帖子评论的艾特内容
    private function topic_create_at(string $content)
    {
        return replace_all_at($content);
    }
}
