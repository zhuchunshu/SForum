<?php

namespace App\Plugins\Core\src\Request\User\Mydata;

use Hyperf\Validation\Request\FormRequest;

class AvatarRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true;
    }

    public function rules(): array
    {
        return [
            "avatar" => "required|image",
        ];
    }
}