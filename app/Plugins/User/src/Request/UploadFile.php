<?php

namespace App\Plugins\User\src\Request;

use Hyperf\Validation\Request\FormRequest;
class UploadFile extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize() : bool
    {
        return true;
    }
    /**
     * Get the validation rules that apply to the request.
     */
    public function rules() : array
    {
        return ["file" => "required|file"];
    }
    public function attributes() : array
    {
        return ["file" => "文件"];
    }
}