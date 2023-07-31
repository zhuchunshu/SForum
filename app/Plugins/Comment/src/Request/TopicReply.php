<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Comment\src\Request;

use Hyperf\Validation\Request\FormRequest;
class TopicReply extends FormRequest
{
    /**
     * Determine if the user is authorized to make this request.
     */
    public function authorize() : bool
    {
        return true;
    }
    public function rules() : array
    {
        return ['comment_id' => 'required|exists:topic_comment,id', 'content' => 'required|string'];
    }
    public function attributes() : array
    {
        return ['comment_id' => '被回复的评论ID', 'content' => '回复内容'];
    }
}