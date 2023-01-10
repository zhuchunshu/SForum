<?php

namespace App\Plugins\Docs\src\Request;

use Hyperf\Validation\Request\FormRequest;

class EditDocsRequest extends FormRequest
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
            "id" => "required|exists:docs,id",
            "html" => "required|string",
            "title" => "required|string",
            "markdown" => "required|string",
        ];
    }

    public function attributes(): array
    {
        return [
            "id" => "文档id",
            "html" => "正文html内容",
            "markdown" => "正文markdown内容",
            "title" => "标题",
        ];
    }
}