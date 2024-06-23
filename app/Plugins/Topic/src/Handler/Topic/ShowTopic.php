<?php

declare (strict_types=1);
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
use App\Plugins\Topic\src\Models\TopicUnlock;
use Hyperf\DbConnection\Db;
class ShowTopic
{
    public function handle($id, $comment_page)
    {
        // 自增浏览量
        go(function () use($id) {
            $updated_at = Topic::query()->where('id', $id)->first()->updated_at;
            Topic::query()->where('id', $id)->increment('view', 1, ['updated_at' => $updated_at]);
        });
        // 自动锁定
        go(function () use($id) {
            $topic = Topic::find($id);
            // 帖子发布天数
            $post_published_time = floor((time() - strtotime((string) $topic->created_at)) / 86400);
            // 判断是否需要锁帖
            if ($topic->status !== 'lock' && get_options('topic_auto_lock') === 'true' && (int) $post_published_time > (int) get_options('topic_auto_lock_day', 30) && !TopicUnlock::where('topic_id',$topic->id)->exists()) {
                // 锁帖
                Db::table('topic')->where('id', $id)->update(['status' => 'lock']);
                // 发送通知
                user_notice()->send($topic->user_id, '你有一条帖子已被系统自动锁定', '帖子《<a href="/' . $topic->id . '.html" >' . $topic->title . '</a>》已被系统自动锁定，原因：帖子发布时间已大于系统设定自动锁帖时间', '/' . $topic->id . '.html', false, 'system');
            }
        });
        // 缓存
        $data = Topic::with('tag', 'user', 'topic_updated', 'likes', 'post', 'post.options', 'comments.user', 'comments.post')->find($id);
        // 举报
        if ($data->status === 'report') {
            // 已被举报并批准
            return view('App::topic.report', ['data' => $data]);
        }
        // 评论分页数据
        if (get_options('comment_show_desc', 'off') === 'true') {
            $CommentOrderBy = 'desc';
        } else {
            $CommentOrderBy = 'asc';
        }
        if (!request()->input('comment_sort')) {
            $comment_sort = $CommentOrderBy;
        } elseif (request()->input('comment_sort') === 'asc' || request()->input('comment_sort') === 'desc') {
            $comment_sort = request()->input('comment_sort');
        } else {
            $comment_sort = $CommentOrderBy;
        }
        $comment = TopicComment::withTrashed()->where(['topic_id' => $id])->with('topic', 'user', 'parent', 'likes', 'post', 'post.options')->orderBy('optimal', 'desc')->orderBy('created_at', $comment_sort)->paginate((int) get_options('comment_page_count', 15));
        // ContentParse data
        $parseData = ['topic' => $data];
        return view('App::topic.show.show', ['data' => $data, 'comment_sort' => $comment_sort, 'comment' => $comment, 'comment_page' => $comment_page, 'parseData' => $parseData]);
    }
}