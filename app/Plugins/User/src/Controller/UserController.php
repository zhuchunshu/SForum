<?php


namespace App\Plugins\User\src\Controller;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserFans;
use App\Plugins\User\src\Models\UsersCollection;
use Hyperf\HttpMessage\Stream\SwooleStream;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Multiavatar;
use Psr\Http\Message\ResponseInterface;
use App\Plugins\User\src\Models\UserClass as UserClassModel;

#[Controller]
class UserController
{
    /**
     * 用户列表
     */
    #[GetMapping(path:"/users")]
    public function list(){
        if(auth()->check()){
	        $count = User::query()->count();
	        $page = User::query()->paginate(20);
	        return view("User::list",['page' => $page,'count'=>$count]);
        }
		return redirect()->url('/')->with('danger','权限不足')->go();
    }

    /**
     * 用户信息
     * @param $username
     * @return ResponseInterface
     */
    #[GetMapping(path:"/users/{id}.html")]
    public function data($id){
        if (! User::query()->where('id', $id)->count()) {
            return admin_abort('页面不存在', 404);
        }
        $user = User::query()->find($id);
        return view('User::data', ['user' => $user]);
    }

    /**
     * 用户信息
     * @param $username
     * @return ResponseInterface
     */
    #[GetMapping(path:"/users/{username}.username")]
    public function username($username){
	    $username = urldecode($username);
        if(!User::query()->where("username",$username)->count()){
            return admin_abort("页面不存在",404);
        }
        $user = User::query()->where("username",$username)->first();
        return redirect()->url('/users/'.$user->id.".username")->go();
    }



    #[GetMapping(path:"/users/group/{id}.html")]
    public function group_data($id): ResponseInterface
    {
        if(!UserClassModel::query()->where("id",$id)->count()){
            return admin_abort("页面不存在","404");
        }
        $userCount = User::query()->where("class_id",$id)->count();
        $data = UserClassModel::query()->where("id",$id)->first();
        $user = User::query()->where("class_id",$id)->paginate(30);
        return view("User::group_data",['userCount' => $userCount,'data' => $data,'user'=>$user]);
    }
	

    // 用户帖子
    #[GetMapping(path:"/users/topic/{username}.html")]
    public function topic($username){
	    $username = urldecode($username);
        if(!User::query()->where("username",$username)->count()){
            return admin_abort("用户名为:".$username."的用户不存在");
        }
        $user = User::query()->where("username",$username)->first();
        $page = Topic::query()->where(['status' => 'publish','user_id' => $user->id])
            ->orderBy('created_at','desc')
            ->paginate(15);
        return view("User::topic",['page' => $page,'user' => $user]);
    }

    // 用户粉丝
    #[GetMapping(path:"/users/fans/{username}.html")]
    public function fans($username){
		$username = urldecode($username);
        if(!User::query()->where("username",$username)->count()){
            return admin_abort("用户名为:".$username."的用户不存在");
        }
        $user = User::query()->where("username",$username)->first();
        $page = UserFans::query()
            ->where("user_id",$user->id)
            ->with('fans')
            ->paginate(15);
        return view("User::fans",['page' => $page,'user' => $user]);
    }

    // 用户收藏
    #[GetMapping(path:"/users/collections/{id}")]
    public function collections($id){
        if(!User::query()->where("id",$id)->exists()){
            return admin_abort("用户不存在",404);
        }
        $quanxian = false;
        if(auth()->id()==$id){
            $quanxian = true;
        }
        $user = User::query()->where("id",$id)->first();
        $page = UsersCollection::query()->where("user_id",$id)->orderBy("id","desc")->paginate(15);
        return view("User::Collections",['page' => $page,'quanxian' => $quanxian,'user' => $user]);
    }
}
