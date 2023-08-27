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
use App\Plugins\Topic\src\Models\ShortcodePaidPost;
use App\Plugins\Topic\src\Models\ShortcodePaidPostOrder;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;

class BuyPosts
{
    // 需要购买的帖子

    #[ShortCodeR(name: 'buy')]
    public function init($match, ShortcodeInterface $shortCode, $data)
    {
        if ((int) $data['topic']['user_id'] === auth()->id()) {
            return $this->buyed($shortCode);
        }
        $post_id = $data['topic']['post_id'];
        // 标记
        return match (ShortcodePaidPostOrder::where(['user_id' => auth()->id(), 'post_id' => $post_id])->exists()) {
            true => $this->buyed($shortCode),
            false => $this->buy($post_id, $shortCode),
        };
    }

    // 已购买
    private function buyed(ShortcodeInterface $shortCode): \Psr\Http\Message\ResponseInterface
    {
        return view('Topic::ShortCode.buy.buyed', [
            'shortCode' => $shortCode,
        ]);
    }

    private function buy(string | int $post_id, ShortcodeInterface $shortCode)
    {
        $amount = $shortCode->getParameter('amount');
        if (! $amount) {
            return $this->alert('amount参数必须');
        }
        // 判断amount 格式
        if (! is_numeric($amount)) {
            return $this->alert('amount参数必须为数字');
        }
        // 判断amount 是否大于或等于0.01
        if ($amount < 0.01) {
            return $this->alert('amount参数必须大于0.01');
        }

        // 判断type
        $type = $shortCode->getParameter('type', 'money');

        if ($type === 'money' || $type === 'credits' || $type === 'golds') {
            // 更新表价格
            $this->set_amount($post_id, $type, (float) $amount);
        } else {
            return $this->alert('type参数错误');
        }
        // 付款
        return $this->pay($post_id, $type, (float) $amount, $shortCode);
    }

    private function alert(string $content): string
    {
        return <<<HTML
<div class="alert alert-danger" role="alert">
    buy短标签使用出错：{$content}
</div>
HTML;
    }

    // 设置价格
    private function set_amount(string | int $post_id, string $type, float $amount): void
    {
        go(function () use ($post_id, $type, $amount) {
            if (ShortcodePaidPost::where('post_id', $post_id)->exists()) {
                ShortcodePaidPost::where('post_id', $post_id)->update([
                    'type' => $type,
                    'amount' => $amount,
                ]);
            } else {
                ShortcodePaidPost::create([
                    'post_id' => $post_id,
                    'type' => $type,
                    'amount' => $amount,
                ]);
            }
        });
    }

    private function pay(int | string $post_id, string $type, float $amount, ShortcodeInterface $shortCode)
    {
        return view('Topic::ShortCode.buy.' . $type, [
            'post_id' => $post_id,
            'amount' => $amount,
            'shortCode' => $shortCode,
        ]);
    }
}
