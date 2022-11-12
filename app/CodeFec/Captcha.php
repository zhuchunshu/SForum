<?php

namespace App\CodeFec;

use Gregwar\Captcha\CaptchaBuilder;
use Gregwar\Captcha\PhraseBuilder;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;

#[Controller]
class Captcha
{
    // 获取验证码
    public function get()
    {
        return '/captcha';
    }
    
    #[GetMapping(path:"/captcha")]
    public function build()
    {
        $phraseBuilder = new PhraseBuilder(5, '0123456789');
        $captcha = new CaptchaBuilder(null, $phraseBuilder);
        $captcha->build();
        // 将验证码的值存储到session中
        session()->set('captcha', strtolower($captcha->getPhrase()));
        // 获得验证码图片二进制数据
        $img_content = $captcha->get();
        return response()->raw($img_content);
    }
    
    public function inline(): string
    {
        $phraseBuilder = new PhraseBuilder(5, '0123456789');
        $captcha = new CaptchaBuilder(null, $phraseBuilder);
        $captcha->build();
        // 将验证码的值存储到session中
        session()->set('captcha', strtolower($captcha->getPhrase()));
        // 获得验证码图片二进制数据
        return $captcha->inline();
    }
    
    public function check($captcha): bool
    {
        $result = (string)strtolower($captcha) === (string)session()->get('captcha');
        if ($result) {
            session()->remove('captcha');
        }
        return $result;
    }
}
