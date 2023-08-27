<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Controllers\Api\ShortCode;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\ShortcodePaidPost;
use App\Plugins\Topic\src\Models\ShortcodePaidPostOrder;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller('/api/topic/shortcode')]
class BuyPost
{
    #[PostMapping(path: 'buy.post')]
    public function request(): array
    {
        if (! auth()->check()) {
            return json_api(401, false, ['msg' => '请先登录']);
        }
        $post_id = request()->input('post_id');
        if (! ShortcodePaidPost::where('post_id', $post_id)->exists()) {
            return json_api(404, false, ['msg' => '未找到此帖子关联的付款项']);
        }

        // 获取付款项信息
        $shortcode = ShortcodePaidPost::where('post_id', $post_id)->first();

        // 判断是否已经购买
        if (ShortcodePaidPostOrder::where(['user_id' => auth()->id(), 'post_id' => $post_id])->exists()) {
            return json_api(403, false, ['msg' => '已经购买过此帖子,请刷新页面']);
        }

        // 获取代币名称
        $coin_name = match ($shortcode->type) {
            'money' => get_options('wealth_money_name', '余额'),
            'credits' => get_options('wealth_credit_name', '积分'),
            'golds' => get_options('wealth_golds_name', '金币'),
        };
        // 判断代币是否足够
        if (auth()->data()->Options->{$shortcode->type} < $shortcode->amount) {
            return json_api(403, false, ['msg' => '你的' . $coin_name . '不足,请充值后再来购买', 'url' => url('/users/' . auth()->id() . '.html?m=users_home_menu_8')]);
        }
        return $this->pay($shortcode, $coin_name);
    }

    private function pay($shortcode, string $coin_name)
    {
        // 扣除代币
        $user = auth()->data();
        $coin = $user->Options->{$shortcode->type};
        UsersOption::where('id', $user->Options->id)->decrement($shortcode->type, $shortcode->amount * 100);

        // 创建订单
        ShortcodePaidPostOrder::create([
            'post_id' => $shortcode->post_id,
            'user_id' => auth()->id(),
        ]);

        // 写购买记录
        create_amount_record($user->id, $coin, $coin - $shortcode->amount, $shortcode->type, -$shortcode->amount, null, '购买帖子隐藏内容,post_id:' . $shortcode->post_id);

        // 把收益发给帖子作者
        $this->give($shortcode);

        return json_api(200, true, ['msg' => '购买成功,扣除' . $coin_name . ': ' . $shortcode->amount]);
    }

    // 把收益发放给帖子作者
    private function give(ShortcodePaidPost $shortcode): void
    {
        // 获取帖子
        $post = Post::find($shortcode->post_id);
        // 获取帖子作者
        $user = $post->user;
        // 获取代币
        $coin = $user->Options->{$shortcode->type};
        // 增加代币
        UsersOption::where('id', $user->Options->id)->increment($shortcode->type, $shortcode->amount * 100);
        // 写收益记录
        create_amount_record($user->id, $coin, $coin + $shortcode->amount, $shortcode->type, $shortcode->amount, null, '帖子隐藏内容收益,post_id:' . $shortcode->post_id);
        $url = url('/' . $post->topic->id . '.html');
        $coin_name = match ($shortcode->type) {
            'money' => get_options('wealth_money_name', '余额'),
            'credits' => get_options('wealth_credit_name', '积分'),
            'golds' => get_options('wealth_golds_name', '金币'),
        };
        // 发送收益通知
        user_notice()->send(
            $user->id,
            '你的帖子付费可见内容为你带来了收益',
            '你的帖子付费可见内容为你带来了收益，帖子id:' . $post->id . '，收益：' . $shortcode->amount . $coin_name . '，已到账',
            $url,
            false
        );
    }
}
