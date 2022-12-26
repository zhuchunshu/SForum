<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Lib\ShortCode;

use App\CodeFec\Annotation\ShortCode\ShortCodeR;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;

class ShortCode
{
    #[ShortCodeR(name: 'only-author')]
    public function onlyAuthor($match, ShortcodeInterface $shortCode, $data)
    {
        if (! @isset($data['comment'])) {
            return '[' . $shortCode->getName() . ']短标签只能用于评论';
        }
        if(auth()->id()==$data['comment']['user_id'] || auth()->id()==$data['comment']['topic']['id']){
            return $shortCode->getContent();
        }
        return <<<HTML
<div class="card card-body disabled">
私密评论，仅作者可见
</div>
HTML;
    }
}
