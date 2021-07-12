<?php

declare(strict_types=1);

namespace App\Plugins\User\src\Request;

use Hyperf\Validation\Request\FormRequest;

class Create extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return true;
    }

    /**
     * Get the validation rules that apply to the request.
     */
    public function rules(): array
    {
        return [
            "name" => "required|string|min:2|max:100|unique:user_class,name",
            "quanxian" => "required|integer|min:1",
            "color" => "required|string",
            "icon" => "nullable"
        ];
    }
    public function attributes(): array
    {
        return [
            "name" => "用户组名称",
            "ename" => "用户组标识",
            "quanxian" => "用户组权限值",
            "color" => "用户组颜色代码"
        ];
    }
}
