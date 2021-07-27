<?php


namespace App\Plugins\Core\src\Request\User;


use Hyperf\Validation\Request\FormRequest;

class UpdateMydataRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            "username" => "nullable|string|unique:users,username,".$this->input("username"),
            "old_pwd" => "nullable|string|min:6|max:20",
            "new_pwd" => "required|string|min:6|max:20",
            "avatar" => "nullable|file"
        ];
    }
}