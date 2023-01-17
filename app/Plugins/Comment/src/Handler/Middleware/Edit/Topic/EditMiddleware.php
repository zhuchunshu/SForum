<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Handler\Middleware\Edit\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;

#[\App\Plugins\Comment\src\Annotation\Topic\UpdateMiddleware]
class EditMiddleware implements MiddlewareInterface
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
                'comment_id' => 'required|exists:topic_comment,id',
                'content' => 'required|string|min:' . get_options('comment_create_min', 1) . '|max:' . get_options('comment_create_max', 200),
            ],
            [],
            [
                'comment_id' => '评论ID',
                'content' => '评论内容',
            ]
        );

        if ($validator->fails()) {
            // Handle exception
            return redirect()->back()->with('danger', $validator->errors()->first())->go();
        }

        $comment = TopicComment::query()->find($data['comment_id']);
        $quanxian = false;
        if (Authority()->check('admin_comment_edit') && auth()->Class()['permission-value'] > curd()->GetUserClass($comment->user->class_id)['permission-value']) {
            $quanxian = true;
        }
        if (Authority()->check('comment_edit') && auth()->id() === $comment->user->id) {
            $quanxian = true;
        }
        if ($quanxian === false) {
            return redirect()->back()->with('danger', '无权修改!')->go();
        }

        if (@$comment->topic->post->options->disable_comment) {
            return redirect()->back()->with('danger', '此帖子关闭了评论功能')->go();
        }
        $data = $this->update($data);
        return $next($data);
    }

    public function update($data)
    {
        // 过滤xss
        $content = xss()->clean($data['content']);

        // 解析艾特
        $content = replace_all_at($content);
        $post_id = TopicComment::query()->find($data['comment_id'])->post_id;
        Post::query()->where('id', $post_id)->update([
            'content' => $content,
        ]);
        return $data;
    }
}
