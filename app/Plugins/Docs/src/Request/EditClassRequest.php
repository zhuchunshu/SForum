<?php

namespace App\Plugins\Docs\src\Request;

use Hyperf\Validation\Request\FormRequest;

class EditClassRequest extends FormRequest
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
            "class_id" => "required|exists:docs_class,id",
            "name" => "required|string|max:25|min:2|unique:docs_class,name,".$this->input("class_id"),
            "userClass" => "nullable|array",
	        'public' => "nullable"
        ];
    }

    public function attributes(): array
    {
        return [
            "class_id" => "文档分类id",
            "name" => "文档名称",
            "icon" => "文档图标",
            "userClass" => "可查看的用户组",
	        "public" => "公开属性"
        ];
    }
}