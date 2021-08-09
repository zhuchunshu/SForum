<?php


namespace App\Plugins\Topic\src\Requests;


use Hyperf\Validation\Request\FormRequest;

class EditTagRequest extends FormRequest
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
            "id" => "integer|exists:topic_tag,id",
            "name" => "required|string|max:25|min:2|unique:topic_tag,name,".$this->input("id").",id",
            "color" => "required|string",
            "description" => "nullable|string"
        ];
    }

    public function attributes(): array
    {
        return [
            "cfpassword" => "Confirm Password"
        ];
    }
}