<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Controller;

use App\Plugins\Comment\src\Model\TopicComment;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller(prefix: '/comment')]
class IndexController
{
    #[GetMapping(path: 'topic/{id}/edit')]
    public function edit_topic_comment($id)
    {
        if (! TopicComment::query()->where('id', $id)->exists()) {
            return admin_abort('id为:' . $id . '的评论不存在', 404);
        }
        $data = TopicComment::query()->where('id', $id)->first();
        $quanxian = false;
        if (Authority()->check('admin_topic_edit') && curd()->GetUserClass(auth()->data()->class_id)['permission-value'] > curd()->GetUserClass($data->user->class_id)['permission-value']) {
            $quanxian = true;
        }
        if (Authority()->check('topic_edit') && auth()->id() === $data->user->id) {
            $quanxian = true;
        }
        if ($quanxian === false) {
            return admin_abort('无权操作!', 419);
        }
        return view('Comment::topic.edit', ['data' => $data]);
    }
}
