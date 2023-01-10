<?php

namespace App\Plugins\Docs\src\Request;

use Hyperf\Validation\Request\FormRequest;

class CreateDocsRequest extends FormRequest
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
            "html" => "required|string",
            "title" => "required|string",
            "markdown" => "required|string",
        ];
    }

    public function attributes(): array
    {
        return [
            "class_id" => "文档分类id",
            "html" => "正文html内容",
            "markdown" => "正文markdown内容",
            "title" => "标题",
        ];
    }
}