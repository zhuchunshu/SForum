<?php

namespace App\Request\Admin;

use Hyperf\Validation\Request\FormRequest;

class ThemeUpload extends FormRequest
{
	/**
	 * Determine if the user is authorized to make this request.
	 */
	public function authorize(): bool
	{
		return true;
	}
	
	/**
	 * Get the validation rules that apply to the request.
	 */
	public function rules(): array
	{
		return [
			'file' => 'mimes:zip,gzip'
		];
	}
	
	public function attributes(): array
	{
		return [
			'file' => '上传的文件'
		];
	}
}