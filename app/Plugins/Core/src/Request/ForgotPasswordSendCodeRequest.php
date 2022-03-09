<?php

namespace App\Plugins\Core\src\Request;

use Hyperf\Validation\Request\FormRequest;

class ForgotPasswordSendCodeRequest extends FormRequest
{
	/**
	 * Determine if the user is authorized to make this request.
	 */
	public function authorize(): bool
	{
		return true;
	}
	
	public function rules(): array
	{
		return [
			"email" => "required|email|exists:users,email",
			'captcha' => 'nullable',
		];
	}
	
	public function attributes(): array
	{
		return [
			'email' => '邮箱',
			'captcha' => '验证码'
		];
	}
}