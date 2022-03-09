<?php

declare(strict_types=1);

namespace App\Plugins\Blog\src\Request;

use Hyperf\Validation\Request\FormRequest;

class ClassEdit extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize(): bool
    {
        return auth()->check();
    }

    /**
     * Get the validation rules that apply to the request.
     */
    public function rules(): array
    {
        return [
			'token' => 'required|exists:blog_class,token',
	        'class_id' => 'required',
	        'name' => 'required'
        ];
    }
}
