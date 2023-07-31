<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Request;

use Hyperf\Validation\Request\FormRequest;
class RegisterRequest extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize() : bool
    {
        return true;
    }
    public function rules() : array
    {
        return ["username" => "required|string|max:25|min:2|alpha_num|unique:users,username", "password" => "required|string|min:8|max:30", "email" => "required|email|unique:users,email", "cfpassword" => "required|min:8|max:30", "captcha" => "nullable", 'invitationCode' => 'nullable'];
    }
    public function attributes() : array
    {
        return ["cfpassword" => "重复密码", 'password' => '密码', 'email' => '邮箱', 'captcha' => '验证码', 'username' => '用户名', 'invitationCode' => '邀请码'];
    }
}