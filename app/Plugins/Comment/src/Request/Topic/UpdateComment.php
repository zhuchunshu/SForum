<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Request\Topic;

use Hyperf\Validation\Request\FormRequest;

class UpdateComment extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            'comment_id' => 'required|exists:topic_comment,id',
            'content' => 'required|string|min:' . get_options('comment_reply_min', 1) . '|max:' . get_options('comment_reply_max', 200),
        ];
    }

    public function attributes(): array
    {
        return [
            'comment_id' => '评论ID',
            'content' => __('topic.comment.comment content'),
        ];
    }
}
