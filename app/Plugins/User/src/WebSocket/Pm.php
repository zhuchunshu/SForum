<?php

namespace App\Plugins\User\src\WebSocket;


use App\Model\AdminOption;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\ContentParse;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersAuth;
use App\Plugins\User\src\Models\UsersPm;
use Hyperf\SocketIOServer\Annotation\Event;
use Hyperf\SocketIOServer\Annotation\SocketIONamespace;
use Hyperf\SocketIOServer\BaseNamespace;
use Hyperf\SocketIOServer\Socket;
use Hyperf\Utils\Codec\Json;
use Hyperf\Utils\Str;

#[SocketIONamespace("/User/Pm")]
class Pm extends BaseNamespace
{
	/**
	 * @param string $data
	 */
	#[Event("event")]
	public function onEvent(Socket $socket, $data)
	{
		// 应答
		return 'Event Received: ' . $data;
	}
	
	/**
	 * @param string $data
	 */
	#[Event("join-room")]
	public function onJoinRoom(Socket $socket, $data)
	{
		return $this->getMsg($socket,$data);
	}
	
	
	
	#[Event('getMsg')]
	public function getMsg(Socket $socket,$data)
	{
		//$socket->emit('getMsg', 'can you hear me?', 1, 2, 'abc');
		$data = Json::decode($data);
		$token = $data['token'];
		$to_id = $data['to_id'];
		if(!UsersAuth::query()->where('token',$token)->exists() || !User::query()->where('id',$to_id)->exists()){
			return ;
		}
		$auth = UsersAuth::query()->where('token',$token)->first()->user;
		UsersPm::query()->where('to_id',$auth->id)->where('from_id',$to_id)->update(['read' => true]);
		\App\Plugins\User\src\Models\UsersPm::query()->where(['from_id'=>$to_id,'to_id' => $auth->id])->update(['read' => true]);
		$msg = UsersPm::query(true)->where([['from_id',$auth->id],['to_id',$to_id]])->Orwhere([['to_id',$auth->id],['from_id',$to_id]])->count();
		return $socket->emit('getMsg', $msg);
	}
	
	#[Event('sendMsg')]
	public function sendMsg(Socket $socket,$data)
	{
		//$socket->emit('getMsg', 'can you hear me?', 1, 2, 'abc');
		$data = Json::decode($data);
		$token = $data['token'];
		$to_id = $data['to_id'];
		if(!UsersAuth::query()->where('token',$token)->exists() || !User::query()->where('id',$to_id)->exists()){
			return ;
		}
		if(Str::length($data['msg'])>$this->get_options('pm_msg_maxlength',300)){
			return $socket->emit('getMsg', '发送失败','长度超出闲置');
		}
		$auth = UsersAuth::query()->where('token',$token)->first()->user;
		if($this->get_user_settings($to_id,'user_message_switch','开启')!=="开启"){
			return $socket->emit('getMsg', '发送失败','用户未开启私信功能');
		}
		UsersPm::query()->create([
			'from_id' => $auth->id,
			'to_id' => $to_id,
			'message' => $data['msg']
		]);
		$msg = UsersPm::query(true)->where([['from_id',$auth->id],['to_id',$to_id]])->Orwhere([['to_id',$auth->id],['from_id',$to_id]])->count();
		return $socket->emit('getMsg', $msg);
		
	}
	
	private function get_user_settings(int|string $user_id, string $name, string $default=""){
		if(!cache()->has('user.settings.'.$user_id.'.'.$name)){
			cache()->set("user.settings.".$user_id.'.'.$name,@\App\Plugins\User\src\Models\UsersSetting::query()->where(["user_id"=>$user_id,"name"=>$name])->first()->value);
		}
		return $this->core_default(cache()->get("user.settings.".$user_id.".".$name),$default);
	}
	
	private function core_default($string=null,$default=null){
		if($string){
			return $string;
		}
		return $default;
	}
	
	private function get_options($name,$default=""){
		if(!cache()->has('admin.options.'.$name)){
			cache()->set("admin.options.".$name,@AdminOption::query()->where("name",$name)->first()->value);
		}
		return $this->core_default(cache()->get("admin.options.".$name),$default);
	}
}