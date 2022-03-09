<?php

namespace App\Plugins\Blog\src\Request;

use Hyperf\Validation\Request\FormRequest;

class ArticleCreate extends FormRequest
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
			'class_id' => 'required|exists:blog_class,id',
			'title' => 'required',
			'content' => 'required',
			'markdown' => 'required'
		];
	}
	
	public function attributes(): array
	{
		return [
			'class_id' => '分类id',
			'title' => '标题',
			'content' => '正文',
			'markdown' => 'markdown内容'
		];
	}
}