<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
use App\Plugins\Comment\src\ContentParse;
use App\Plugins\Comment\src\Model\TopicComment;

if (! function_exists('get_topic_comment_page')) {
    /**
     * 获取评论所在的页码
     */
    function get_topic_comment_page(string|int $comment_id): int
    {
        if (! \App\Plugins\Comment\src\Model\TopicComment::query()->where('id', $comment_id)->exists()) {
            return 1;
        }
        // 所在帖子ID
        $topic_id = \App\Plugins\Comment\src\Model\TopicComment::query()->where('id', $comment_id)->value('topic_id');
        // 每页加载的评论数量
        $comment_num = get_options('comment_page_count', 15);
        $inPage = 1;
        // 获取最后一页页码
        $lastPage = TopicComment::query()
            ->where(['topic_id' => $topic_id])
            ->paginate((int) $comment_num)->lastPage();
        if (get_options('comment_show_desc', 'off') === 'true') {
            $CommentOrderBy = 'desc';
        } else {
            $CommentOrderBy = 'asc';
        }
        $comment_sort = $CommentOrderBy;
        for ($i = 0; $i < $lastPage; ++$i) {
            $page = $i + 1;
            $data = TopicComment::query()
                ->where(['topic_id' => $topic_id])
                ->with('topic', 'user', 'parent')
                ->orderBy('optimal', 'desc')
                ->orderBy('created_at', $comment_sort)
                ->paginate((int) $comment_num, ['*'], 'page', $page)->items();
            foreach ($data as $value) {
                if ((int) $value->id === (int) $comment_id) {
                    $inPage = $page;
                }
            }
        }
        return $inPage;
    }
}

if (! function_exists('get_topic_comment_floor')) {
    /**
     * 获取评论楼层
     * @return int
     */
    function get_topic_comment_floor(int $comment_id): ?int
    {
        $floor = null;
        if (! \App\Plugins\Comment\src\Model\TopicComment::query()->where('id', $comment_id)->exists()) {
            $floor = null;
        }
        // 所在帖子ID
        $topic_id = \App\Plugins\Comment\src\Model\TopicComment::query()->where('id', $comment_id)->value('topic_id');
        // 每页加载的评论数量
        $comment_num = get_options('comment_page_count', 15);
        $comment_page = get_topic_comment_page($comment_id);
        // ($key + 1)+(($comment->currentPage()-1)*get_options('comment_page_count',15))
        $page = TopicComment::query()
            ->where(['topic_id' => $topic_id])
            ->paginate((int) $comment_num, ['*'], 'page', $comment_page);
        foreach ($page as $k => $v) {
            if ((int) $v->id === $comment_id) {
                $floor = ($k + 1) + (($comment_page - 1) * get_options('comment_page_count', 15));
            }
        }
        return $floor;
    }
}

if (! function_exists('get_topic_comment')) {
    function get_topic_comment($id): array | \Hyperf\Database\Model\Collection | \Hyperf\Database\Model\Model | bool | TopicComment
    {
        $comment = TopicComment::find($id);
        if (! $comment) {
            return false;
        }
        return $comment;
    }
}

if (! function_exists('CommentContentParse')) {
    function CommentContentParse(): ContentParse
    {
        return new ContentParse();
    }
}

if (! function_exists('get_topic_comment_url')) {
    // 获取评论链接
    function get_topic_comment_url($comment_id, $link = false): string | null
    {
        $comment = get_topic_comment($comment_id);
        if (! $comment) {
            return null;
        }
        $url = '/' . $comment->topic_id . '.html/' . $comment_id . '?page=' . get_topic_comment_page($comment_id);
        if ($link === true) {
            $url = url($url);
        }
        return $url;
    }
}
