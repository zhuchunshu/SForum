<?php

namespace App\Plugins\Docs\src\Request;

use Hyperf\Validation\Request\FormRequest;

class CreateClassRequest extends FormRequest
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
            "name" => "required|string|max:25|min:2|unique:docs_class,name",
            "userClass" => "nullable|array"
        ];
    }

    public function attributes(): array
    {
        return [
            "name" => "文档名称",
            "userClass" => "可查看的用户组"
        ];
    }
}