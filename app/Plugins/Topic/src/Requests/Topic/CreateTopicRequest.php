<?php

namespace App\Plugins\Topic\src\Requests\Topic;

use Hyperf\Validation\Request\FormRequest;
class CreateTopicRequest extends FormRequest
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
        return ["html" => "required|string|min:" . get_options("topic_create_content_min", 10), "title" => "required|string|min:" . get_options("topic_create_title_min", 1) . "|max:" . get_options("topic_create_title_max", 200), "tag" => "required|exists:topic_tag,id"];
    }
    public function attributes() : array
    {
        return ["html" => "正文html内容", "title" => "标题", "tag" => "板块id"];
    }
}