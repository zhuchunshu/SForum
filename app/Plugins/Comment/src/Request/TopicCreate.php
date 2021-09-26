<?php

namespace App\Plugins\Comment\src\Request;

use Hyperf\Validation\Request\FormRequest;

class TopicCreate extends FormRequest
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
            "topic_id" => "required|exists:topic,id",
            "content" => "required|string|min:".get_options("comment_create_min",1)."|max:".get_options("comment_create_max",200),
            "markdown" => "required|string",
        ];
    }

    public function attributes(): array
    {
        return [
            "topic_id" => "帖子ID",
            "html" => "评论内容",
            "markdown" => "评论markdown内容"
        ];
    }
}