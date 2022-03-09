<?php

namespace App\Plugins\Core\src\Request;

use Hyperf\Validation\Request\FormRequest;

class ForgotPassword extends FormRequest
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
			'code' => 'nullable',
			'password' => "required|min:8|max:30",
			'cfpassword' => 'required|min:8|max:30'
		];
	}
	
	public function attributes(): array
	{
		return [
			'email' => '邮箱',
			'code' => '验证码',
			'password' => '新密码',
			'cfpassword' =>'重复密码'
		];
	}
}