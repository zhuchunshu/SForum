<?php


namespace App\Plugins\Core\src\Request\User\Mydata;


use Hyperf\Validation\Request\FormRequest;
use JetBrains\PhpStorm\ArrayShape;

class JibenRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

   public function rules(): array
    {
        return [
            "old_pwd" => "nullable|string|min:6|max:20",
            "new_pwd" => "nullable|string|min:6|max:20",
        ];
    }
    public function attributes(): array
    {
        return [
            "old_pwd" => "旧密码",
            "new_pwd" => "新密码"
        ];
    }
}