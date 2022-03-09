<?php
namespace App\Plugins\Core\src\Controller;

use App\Plugins\Core\src\Models\InvitationCode;
use App\Plugins\Core\src\Request\ForgotPassword;
use App\Plugins\Core\src\Request\ForgotPasswordSendCodeRequest;
use App\Plugins\Core\src\Request\LoginRequest;
use App\Plugins\Core\src\Request\LoginUsernameRequest;
use App\Plugins\Core\src\Request\RegisterRequest;
use App\Plugins\User\src\Event\AfterRegister;
use App\Plugins\User\src\Middleware\LoginMiddleware;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersOption;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\Middleware;
use Hyperf\HttpServer\Annotation\PostMapping;
use HyperfExt\Hashing\Hash;
use Illuminate\Support\Str;

#[Controller()]
#[Middleware(\App\Plugins\User\src\Middleware\AuthMiddleware::class)]
class UserController
{

    #[GetMapping(path:"/login")]
    public function login(): \Psr\Http\Message\ResponseInterface
    {

        return view("Core::user.sign",['title' => "登陆","view" => "Core::user.login"]);
    }
	
	// 忘记密码
	#[GetMapping(path:"/forgot-password")]
	public function forgot_password(): \Psr\Http\Message\ResponseInterface
	{
		
		return view("Core::user.sign",['title' => "找回密码","view" => "Core::user.forgot_password"]);
	}
	
	#[PostMapping(path:"/forgot-password")]
	public function forgot_password_submit(ForgotPassword $request)
	{
		$data = $request->validated();
		$email = $data['email'];
		$code = $data['code'];
		$password = $data['password'];
		$cfpassword = $data['cfpassword'];
		if(!cache()->has('forgot-password.'.$code)){
			return Json_Api(403,false,['msg' => '验证失败,可能是验证码已过期!']);
		}
		$code_email = cache()->get('forgot-password.'.$code);
		if($email!==$code_email){
			return Json_Api(403,false,['msg' => '邮箱核验失败!']);
		}
		if($password!==$cfpassword){
			return Json_Api(403,false,['msg' => '两次输入密码不一致!']);
		}
		
		User::query()->where('email',$email)->update([
			'password' => Hash::make($data['password'])
		]);
		cache()->delete('forgot-password.'.$code);
		
		auth()->SignIn($email,$password);
		
		return Json_Api(200,true,['msg' => '修改成功!']);
	}
	
	// 找回密码 -- 发送验证码
	#[PostMapping(path:"/forgot-password/sendCode")]
	public function forgot_password_sendCode(ForgotPasswordSendCodeRequest $request){
		$data = $request->validated();
		$email = $data['email'];
		$captcha = $data['captcha'];
		if(!captcha()->check($captcha)){
			return Json_Api(403,false,['msg' => '验证码错误!']);
		}
		// 生成验证码
		$code = Str::random(6);
		cache()->set('forgot-password.'.$code,$email,600);
		// 发送邮件
		$mail = Email();
		go(function() use ($email,$code,$mail){
			$mail->addAddress($email);
			$mail->Subject = "【".get_options("web_name")."】 请查看你的验证码!";
			$mail->Body    = <<<HTML
<h3>你正在尝试找回密码</h3>
<p>验证码为: <code>{$code}</code> </p>
<p>有效期 10分钟</p>
HTML;
			$mail->send();
		});
		return Json_Api(200,true,['msg' => '已发送验证码!']);
	}
	
	// 找回密码 -- 验证
	#[PostMapping(path:"/forgot-password/verifyCode")]
	public function forgot_password_verifyCode(){
		$code = request()->input('code');
		if(!$code){
			return Json_Api(403,false,['msg' => '请求参数不足!']);
		}
		if(!cache()->has('forgot-password.'.$code)){
			return Json_Api(403,false,['msg' => '验证失败,可能是验证码已过期!']);
		}
		
		return Json_Api(200,true,['msg' => '验证成功! 请设置新密码']);
	}
	
	#[GetMapping(path:"/login/username")]
	public function login_username(): \Psr\Http\Message\ResponseInterface
	{
		
		return view("Core::user.sign",['title' => "使用用户名登陆","view" => "Core::user.login_username"]);
	}

    #[GetMapping(path:"/register")]
    public function register(): \Psr\Http\Message\ResponseInterface
    {
        if(get_options("core_user_reg_switch","开启")==="关闭"){
            return admin_abort("注册功能已关闭",403);
        }
        return view("Core::user.sign",['title' => "注册","view" => "Core::user.register"]);
    }

    #[PostMapping(path: "/register")]
    public function register_post(RegisterRequest $request): array
    {
        if(get_options("core_user_reg_switch","开启")==="关闭"){
            return Json_Api(403,false,['msg' => "注册功能已关闭"]);
        }
        $data = $request->validated();
		// 定义变量
	    $password = $data['password'];
		$cfpassword = $data['cfpassword'];
		$captcha = $data['captcha'];
		$email = $data['email'];
		$username = $data['username'];
		$invitationCode = $data['invitationCode'];
	
		// 判断验证码问题
	    if((get_options('core_user_reg_captcha', '开启') === '开启') && !captcha()->check($captcha)) {
		    return Json_Api(403,false,['msg' => '验证码错误!']);
	    }
		
		// 判断两次输入密码是否一致
        if($password !== $cfpassword){
            return Json_Api(403,false,['msg' => "两次输入密码不一致"]);
        }
	
	    // 判断验证码问题
	    if((get_options('core_user_reg_yaoqing', '关闭') === '开启') && !InvitationCode::query()->where(['code' => $invitationCode,'status' => false])->exists()) {
		    return Json_Api(403,false,['msg' => '邀请码不存在!']);
	    }
		
        
        $userOption = UsersOption::query()->create(["qianming" => "这个人没有签名"]);
        $result = User::query()->create([
            "username" => $username,
            "email" => $email,
            "password" => Hash::make($password),
            "class_id" => get_options("core_user_reg_defuc",1),
            "_token" => Str::random(),
            "options_id" => $userOption->id
        ]);
	
		// 设置邀请码失效
	    InvitationCode::query()->where(['code' => $invitationCode,'status' => false])->update([
			'user_id' => $result->id,
		    'status' => true
	    ]);
		
		EventDispatcher()->dispatch(new AfterRegister($result));
		// 自动登陆
	    auth()->SignIn($email,$password);
		
        return Json_Api(200,true,['msg' => '注册成功!']);
    }

    #[PostMapping(path: "/login")]
    public function login_post(LoginRequest $request): ?array
    {
        $data = $request->validated();
        $email = $data['email'];
        $password = $data['password'];
		$captcha = $data['captcha'];
	
	    if((get_options('core_user_login_captcha', '开启') === '开启') && !captcha()->check($captcha)) {
		    return Json_Api(403,false,['msg' => '验证码错误!']);
	    }
		
		
        if(auth()->SignIn($email,$password)){
            return Json_Api(200,true,['msg' => '登陆成功!']);
        }

        return Json_Api(403,false,['msg' => '登陆失败,账号或密码错误']);
    }
	
	
	#[PostMapping(path: "/login/username")]
	public function login_username_post(LoginUsernameRequest $request): ?array
	{
		$data = $request->validated();
		$username = $data['username'];
		$password = $data['password'];
		$captcha = $data['captcha'];
		
		if((get_options('core_user_login_captcha', '开启') === '开启') && !captcha()->check($captcha)) {
			return Json_Api(403,false,['msg' => '验证码错误!']);
		}
		
		
		if(auth()->SignInUsername($username,$password)){
			return Json_Api(200,true,['msg' => '登陆成功!']);
		}
		
		return Json_Api(403,false,['msg' => '登陆失败,账号或密码错误']);
	}

    #[PostMapping(path: "/logout")]
    public function logout(): array
    {
        auth()->logout();
        return Json_Api(200,true,['msg' => '退出登陆成功!','url' => '/login']);
    }


}