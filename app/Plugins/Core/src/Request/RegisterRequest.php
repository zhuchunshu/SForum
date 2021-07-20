<?php
declare(strict_types=1);

namespace App\Plugins\Core\src\Request;


use Hyperf\Validation\Request\FormRequest;

class RegisterRequest extends FormRequest
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
            "username" => "required|max:25|min:2|unique:users,username",
            "password" => "required|min:8|max:30",
            "email" => "required|email|unique:users,email",
            "cfpassword" => "required|min:8|max:30",
            "captcha" => "required|integer"
        ];
    }

    public function attributes(): array
    {
        return [
            "cfpassword" => "Confirm Password"
        ];
    }
}