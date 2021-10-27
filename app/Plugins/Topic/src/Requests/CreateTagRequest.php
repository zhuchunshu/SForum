<?php


namespace App\Plugins\Topic\src\Requests;


use Hyperf\Validation\Request\FormRequest;

class CreateTagRequest extends FormRequest
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
            "name" => "required|string|max:25|min:2|unique:topic_tag,name",
            "icon" => "required|image",
            "color" => "required|string",
            'userClass' => 'nullable|array',
            "description" => "nullable|string"
        ];
    }

    public function attributes(): array
    {
        return [
            "name" => "名称",
            "icon" => "图标",
            "color" => "颜色值",
            'userClass' => '可以使用此标签的用户组',
            "description" => "描述"
        ];
    }
}