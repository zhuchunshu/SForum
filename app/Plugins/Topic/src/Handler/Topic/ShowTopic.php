<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Report;
use App\Plugins\Topic\src\Models\Topic;

class ShowTopic
{
    public function handle($id, $comment_page)
    {
        if (Report::query()->where(['type' => 'topic', '_id' => $id, 'status' => 'approve'])->exists()) {
            return admin_abort('此帖子已被举报并批准,无法查看', 403);
        }
        // 自增浏览量
        go(function () use ($id) {
            $updated_at = Topic::query()->where('id', $id)->first()->updated_at;
            Topic::query()->where('id', $id)->increment('view', 1, ['updated_at' => $updated_at]);
        });

        // 缓存
        $data = Topic::query()->with('tag', 'user', 'topic_updated', 'likes', 'post', 'post.options', 'comments.user', 'comments.post')->find($id);
        // 评论分页数据
        if (get_options('comment_show_desc', 'off') === 'true') {
            $CommentOrderBy = 'desc';
        } else {
            $CommentOrderBy = 'asc';
        }
        if (! request()->input('comment_sort')) {
            $comment_sort = $CommentOrderBy;
        } elseif (request()->input('comment_sort') === 'asc' || request()->input('comment_sort') === 'desc') {
            $comment_sort = request()->input('comment_sort');
        } else {
            $comment_sort = $CommentOrderBy;
        }
        $comment = TopicComment::query()
            ->where(['status' => 'publish', 'topic_id' => $id])
            ->with('topic', 'user', 'parent', 'likes', 'post', 'post.options')
            ->orderBy('optimal', 'desc')
            ->orderBy('created_at', $comment_sort)
            ->paginate((int) get_options('comment_page_count', 15));
        // ContentParse data
        $parseData = [
            'topic' => $data,
        ];
        return view('App::topic.show.show', ['data' => $data,  'comment_sort' => $comment_sort, 'comment' => $comment, 'comment_page' => $comment_page, 'parseData' => $parseData]);
    }
}
