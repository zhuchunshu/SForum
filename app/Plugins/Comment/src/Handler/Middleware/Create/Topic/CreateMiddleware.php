<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Create\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;

#[\App\Plugins\Comment\src\Annotation\Topic\CreateMiddleware]
class CreateMiddleware implements MiddlewareInterface
{
    /**
     * @Inject
     */
    protected ValidatorFactoryInterface $validationFactory;

    public function handler($data, \Closure $next)
    {
        $validator = $this->validationFactory->make(
            $data,
            [
                'topic_id' => 'required|exists:topic,id',
                'content' => 'required|string|min:' . get_options('comment_create_min', 1) . '|max:' . get_options('comment_create_max', 200),
            ],
            [],
            [
                'topic_id' => '帖子ID',
                'content' => '评论内容',
            ]
        );

        if ($validator->fails()) {
            // Handle exception
            cache()->delete('comment_create_time_' . auth()->id());
            return redirect()->back()->with('danger', $validator->errors()->first())->go();
        }
        $topic = Topic::query()->find($data['topic_id']);
        if (@$topic->post->options->disable_comment) {
            cache()->delete('comment_create_time_' . auth()->id());
            return redirect()->back()->with('danger', '此帖子关闭了评论功能')->go();
        }
        $data = $this->create($data);
        return $next($data);
    }

    public function create($data)
    {
        // 原内容
        $_content = $data['content'];

        // 过滤xss
        $content = xss()->clean($data['content']);

        // 解析艾特
        $content = replace_all_at($content);

        $post = Post::query()->create([
            'content' => $content,
            'user_agent' => get_user_agent(),
            'user_ip' => get_client_ip(),
            'user_id' => auth()->id(),
        ]);
        $comment = TopicComment::query()->create([
            'topic_id' => $data['topic_id'],
            'post_id' => $post->id,
            'user_id' => auth()->id(),
        ]);
        // 给posts表设置comment_id字段的值
        $posts = Post::query()->where('id', $post->id)->update(['comment_id' => $comment->id]);

        // 艾特被回复的人
        $this->at_user($comment, $_content);

        //发布成功
        // 发送通知
        $topic_data = Topic::query()->find($data['topic_id']);
        if ($topic_data->user_id != auth()->id()) {
            $title = auth()->data()->username . '评论了你发布的帖子!';
            $content = view('Comment::Notice.comment', ['comment' => $content, 'user_data' => auth()->data(), 'data' => $comment]);
            $action = '/' . $topic_data->id . '.html';
            user_notice()->send($topic_data->user_id, $title, $content, $action);
        }

        $data['comment'] = $comment;
        $data['posts'] = $posts;
        return $data;
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
                            '/' . $data->topic_id . '.html/' . $data->id . '?page=' . get_topic_comment_page($data->id)
                        );
                    }
                }
            });
        }
    }
}
