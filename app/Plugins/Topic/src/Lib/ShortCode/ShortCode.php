<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
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
        if (auth()->id() == $data['comment']['user_id'] || auth()->id() == $data['comment']['topic']['user_id']) {
            return $shortCode->getContent();
        }
        return <<<'HTML'
<div class="card card-body disabled">
私密评论，仅楼主可见
</div>
HTML;
    }

    #[ShortCodeR(name: 'code')]
    public function code($match, ShortcodeInterface $shortCode)
    {
        $lang = $shortCode->getParameter('lang','text');
        $content = $shortCode->getContent();
        $content = str_replace("<p>", "", $content);
        $content = str_replace("</p>", "", $content);
        $content = str_replace("<br>", "\n", $content);
        $content = trim($content);
        return <<<HTML
<pre class="language-{$lang}"><code>{$content}</code></pre>
HTML;

    }
}
