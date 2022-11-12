<?php

namespace App\CodeFec\Ui\Generate;

use Hyperf\Utils\Str;

class PayGenerate
{

    /**
     * 根据不同状态，选择不同文字样式
     * @param string $status
     * @return string
     */
    public function status_class_text(string $status){
        $style = ['text-blue'];
        // 待支付
        if (Str::is('*待支付*','*'.$status.'*') || Str::is('*未支付*','*'.$status.'*') || Str::is('*未付款*','*'.$status.'*')){
            $style = ['text-lime'];
        }

        // 支付成功
        if (Str::is('*成功*','*'.$status.'*')){
            $style = ['text-azure'];
        }

        // 退款
        if (Str::is('*退款*','*'.$status.'*')){
            $style = ['text-orange'];
        }

        // 交易取消
        if (Str::is('*取消*','*'.$status.'*')){
            $style = ['text-purple'];
        }

        // 交易关闭
        if (Str::is('*关闭*','*'.$status.'*')){
            $style = ['text-yellow'];
        }

        // 支付失败
        if (Str::is('*失败*','*'.$status.'*')){
            $style = ['text-red'];
        }

        return implode(' ',$style);
    }

    /**
     * 根据不同状态，选择不同颜色名
     * @param string $status
     * @return string
     */
    public function status_color_name(string $status){
        $style = ['blue'];
        // 待支付
        if (Str::is('待支付','*'.$status.'*') || Str::is('*未支付*','*'.$status.'*') || Str::is('*未付款*','*'.$status.'*')){
            $style = ['lime'];
        }

        // 支付成功
        if (Str::is('*成功*','*'.$status.'*',)){
            $style = ['azure'];
        }

        // 退款
        if (Str::is('*退款*','*'.$status.'*',)){
            $style = ['orange'];
        }


        // 交易关闭
        if (Str::is('*关闭*','*'.$status.'*')){
            $style = ['yellow'];
        }

        // 交易取消
        if (Str::is('*取消*','*'.$status.'*')){
            $style = ['purple'];
        }

        // 支付失败
        if (Str::is('*失败*','*'.$status.'*')){
            $style = ['red'];
        }

        return implode(' ',$style);
    }
}