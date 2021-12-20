<?php

namespace App\Plugins\Topic\src\Requests\Topic;

use Hyperf\Validation\Request\FormRequest;

class CreateTopicRequest extends FormRequest
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
            "options_hidden_user_list" => "nullable|string",
            "options_hidden_user_class" => "nullable|string",
            "options_hidden_type" => "nullable|string",
            "html" => "required|string|min:".get_options("topic_create_content_min",10),
            "title" => "required|string|min:".get_options("topic_create_title_min",1)."|max:".get_options("topic_create_title_max",200),
            "markdown" => "required|string|min:".get_options("topic_create_content_min",10),
            "tag" => "required|exists:topic_tag,id"
        ];
    }

    public function attributes(): array
    {
        return [
            "options_hidden_user_list" => "阅读权限 - 隐藏 - 指定用户列表",
            "options_hidden_user_class" => "阅读权限 - 隐藏 - 指定用户组列表",
            "options_hidden_type" => "阅读权限 - 隐藏类型",
            "html" => "正文html内容",
            "markdown" => "正文markdown内容",
            "title" => "标题",
            "tag" => "标签id"
        ];
    }
}