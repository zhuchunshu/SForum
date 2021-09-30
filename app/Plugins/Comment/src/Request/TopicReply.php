<?php

namespace App\Plugins\Comment\src\Request;

use Hyperf\Validation\Request\FormRequest;

class TopicReply extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return true;
    }

    public function rules():array
    {
        return [
            "comment_id" => "required|exists:topic_comment,id",
            "content" => "required|string|min:".get_options("comment_reply_min",1)."|max:".get_options("comment_reply_max",200),
            "markdown" => "required|string",
            "parent_url" => "required",
        ];
    }

    public function attributes(): array
    {
        return [
            "comment_id" => "被回复的评论ID",
            "html" => "评论内容",
            "markdown" => "评论markdown内容",
            "parent_url" => "被回复的帖子链接",
        ];
    }
}