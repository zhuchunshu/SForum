<?php

namespace App\Plugins\Core\src\Request;

use Hyperf\Validation\Request\FormRequest;
class LoginUsernameRequest extends FormRequest
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
        return ["username" => "required|string|exists:users,username", "password" => "required|string", 'captcha' => 'nullable'];
    }
    public function attributes() : array
    {
        return ['username' => '用户名', 'password' => '密码', 'captcha' => '验证码'];
    }
}