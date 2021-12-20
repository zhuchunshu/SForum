<?php

namespace App\Plugins\Core\src\Request\Report;

use Hyperf\Validation\Request\FormRequest;

class CreateRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return auth()->check();
    }

    public function rules(): array
    {
        return [
            "report_reason" => "required|string",
            "type" => "required|string",
            "type_id" => "required|numeric",
            "title" => 'required|string',
            'content' => 'required|string',
            'url' => 'required|string'
        ];
    }

    public function attributes(): array{
        return [
            "report_reason" => "举报原因",
            'title' => '标题',
            'content' => '详细说明',
            'url' => '相关链接'
        ];
    }
}