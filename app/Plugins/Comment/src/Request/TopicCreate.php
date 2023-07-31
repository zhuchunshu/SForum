<?php

namespace App\Plugins\Comment\src\Request;

use Hyperf\Validation\Request\FormRequest;
class TopicCreate extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize() : bool
    {
        return true;
    }
    public function rules() : array
    {
        return ["topic_id" => "required|exists:topic,id", "content" => "required|string"];
    }
    public function attributes() : array
    {
        return ["topic_id" => "帖子ID", "html" => __("topic.comment.comment content")];
    }
}