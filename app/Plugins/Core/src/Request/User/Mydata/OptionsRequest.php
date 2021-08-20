<?php


namespace App\Plugins\Core\src\Request\User\Mydata;


use Hyperf\Validation\Request\FormRequest;

class OptionsRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            "qianming" => "nullable|string|max:1000",
            "qq" => "nullable|numeric|min:5|max:10",
            "wx" => "nullable|max:100",
            "website" => "nullable|url",
            "email" => "nullable|email"
        ];
    }
    public function attributes(): array
    {
        return [
            "qianming" => "签名",
            "wx" => "微信",
            "website" => "个人网站",
            "email" => "展示邮箱"
        ];
    }
}