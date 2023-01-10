<?php

namespace App\Plugins\Docs\src\Controllers;

use App\Plugins\Core\src\Handler\UploadHandler;
use App\Plugins\Docs\src\Model\Docs;
use App\Plugins\Docs\src\Model\DocsClass;
use App\Plugins\Docs\src\Request\CreateClassRequest;
use App\Plugins\Docs\src\Request\EditClassRequest;
use App\Plugins\User\src\Models\UserClass;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;

#[Controller(prefix:"/docs")]
class IndexController
{
    #[GetMapping(path:"")]
    public function index()
    {
	    [$docs] = $this->docsExtracted();
		[$docsAll] = $this->docsAllExtracted();
	    return view("Docs::index",['docs' => $docs,'docsAll'=>$docsAll]);
    }

    #[GetMapping(path:"create.class")]
    public function create_class(){
        if(!Authority()->check("docs_create")){
            return admin_abort('你所在的用户组无权创建文档',401);
        }
        $userClass = UserClass::query()->select('id','name','permission-value')->get();
        return view("Docs::create_class",['userClass' => $userClass]);
    }

    #[PostMapping(path:"create.class")]
    public function create_class_submit(CreateClassRequest $request,UploadHandler $uploader){
        if(!Authority()->check("docs_create")){
            return redirect()->back()->with("danger",'你所在的用户组无权创建文档')->go();
        }
        $quanxian = json_encode($request->input('userClass'));
	    if($request->input('public',false)=="on"){
		    $public = true;
	    }else{
		    $public = false;
	    }
        DocsClass::query()->create([
            'name' => $request->input('name'),
            'icon' => '.',
            'user_id' => auth()->id(),
            'quanxian' => $quanxian,
	        'public' => $public
        ]);
        return redirect()->url('/docs')->with("success",'创建成功!')->go();
    }

    #[GetMapping(path:"create/{id}")]
    public function create($id){
        if(!DocsClass::query()->where('id',$id)->exists()){
            return admin_abort("页面不存在",404);
        }
        $user_id = (int)DocsClass::query()->where('id',$id)->first()->user_id;
        if(auth()->id()!==$user_id || !Authority()->check("docs_create")){
            return admin_abort("无权限",401);
        }
        $data = DocsClass::query()->where('id',$id)->first();
        return view("Docs::create",['data' => $data]);
    }

    #[GetMapping(path:"edit/{id}")]
    public function edit($id)
    {
        if (!Docs::query()->where('id', $id)->exists()) {
            return admin_abort("页面不存在", 404);
        }
        $user_id = (int)Docs::query()->where('id', $id)->first()->user_id;
        $quanxian = false;
        if (auth()->id() === $user_id || Authority()->check("docs_edit")) {
            $quanxian = true;
        }

        if (Authority()->check("admin_docs_edit")) {
            $quanxian = true;
        }

        if (!$quanxian) {
            return admin_abort("无权限", 401);
        }
        $data = Docs::query()->where('id',$id)->first();
        return view("Docs::edit",['data' => $data]);
    }

    #[GetMapping(path:"editClass/{id}")]
    public function edit_class($id){
        if(!DocsClass::query()->where('id',$id)->exists()){
            return admin_abort("页面不存在",404);
        }
        $user_id = (int)DocsClass::query()->where('id',$id)->first()->user_id;
        $quanxian = false;
        if (auth()->id()===$user_id || Authority()->check("docs_edit")) {
            $quanxian = true;
        }

        if(Authority()->check("admin_docs_edit")) {
            $quanxian = true;
        }

        if(!$quanxian){
            return admin_abort("无权限",401);
        }

        $userClass = UserClass::query()->select('id','name','permission-value')->get();
        $data = DocsClass::query()->where('id',$id)->first();
        return view("Docs::edit_class",['data' => $data,'userClass' => $userClass]);
    }

    #[PostMapping(path:"editClass")]
    public function edit_class_submit(EditClassRequest $request){
        $user_id = (int)DocsClass::query()->where('id',$request->input('class_id'))->first()->user_id;
        $quanxian = false;
        if (auth()->id()===$user_id || Authority()->check("docs_edit")) {
            $quanxian = true;
        }

        if(Authority()->check("admin_docs_edit")) {
            $quanxian = true;
        }
        if(!$quanxian){
            return redirect()->url('/docs/editClass/'.$request->input('class_id'))->with("danger",'你所在的用户组无权修改文档')->go();
        }
        $quanxian = json_encode($request->input('userClass'));
		if($request->input('public',false)=="on"){
			$public = true;
		}else{
			$public = false;
		}
        DocsClass::query()->where("id",$request->input("class_id"))->update([
            'name' => $request->input('name'),
            'user_id' => auth()->id(),
            'quanxian' => $quanxian,
	        'public' => $public
        ]);
        return redirect()->url('/docs/editClass/'.$request->input('class_id'))->with("success",'修改成功!')->go();
    }

    #[GetMapping(path:"{id}")]
    public function show($id){
        if(!DocsClass::query()->where('id',$id)->exists()){
            return admin_abort('页面不存在',404);
        }
        $data = DocsClass::query()->where('id',$id)->first();
        $quanxian = false;
        if($data->quanxian!=="null"){
            $p_quanxian = true;
            $arr = json_decode($data->quanxian, true, 512, JSON_THROW_ON_ERROR);
            if(auth()->check() && @in_array(auth()->data()->class_id, $arr)){
                $quanxian = true;
            }
            foreach (UserClass::query()->get() as $value){
                if(!in_array((int)$value->id, $arr)){
                    $p_quanxian = false;
                }
            }
        }else{
            $quanxian = true;
        }


        if((int)$data->user_id === auth()->id()){
            $quanxian = true;
        }
		
		if(DocsClass::query()->where(['id'=>$id,'public' => true])->exists()){
			$quanxian = true;
		}



        if(!$quanxian && !$p_quanxian){
            return admin_abort('无权查看',401);
        }
		
        $docsT = Docs::query()->where("class_id",$id)->select('id','title','class_id')->get();
		if(count($docsT)){
			return redirect()->url('/docs/'.$id.'/'.Docs::query()->where("class_id",$id)->select('id','title','class_id')->first()->id.'.html')->go();
		}
	    [$docs] = $this->docsExtracted();
		[$docsAll] = $this->docsAllExtracted();
	    return view("Docs::show",['data' => $data,'docs' => $docs,'docsT' => $docsT,'docsAll'=>$docsAll]);
    }

    #[GetMapping(path:"/docs/{class_id}/{id}.html")]
    public function showDocs($class_id,$id){
        if(!DocsClass::query()->where('id',$class_id)->exists()){
            return admin_abort('页面不存在',404);
        }
        $data = DocsClass::query()->where('id',$class_id)->first();
        $quanxian = false;
        if($data->quanxian!=="null"){
            $arr = json_decode($data->quanxian, true, 512, JSON_THROW_ON_ERROR);
            if(in_array(auth()->data()->class_id,$arr)){
                $quanxian = true;
            }
        }else{
            $quanxian = true;
        }

        if((int)$data->user_id === auth()->id()){
            $quanxian = true;
        }
	    if(DocsClass::query()->where(['id'=>$id,'public' => true])->exists()){
		    $quanxian = true;
	    }

        if($data->quanxian!=="null"){
            $p_quanxian = true;
            foreach (UserClass::query()->get() as $value){
                if(!in_array((int)$value->id, $arr)){
                    $p_quanxian = false;
                }
            }
        }else{
            $quanxian = true;
        }
        if(!$quanxian && !$p_quanxian){
            return admin_abort('无权查看',401);
        }

        if(!Docs::query()->where(['id' => $id,'class_id' => $class_id])->exists()){
            return admin_abort('页面不存在',404);
        }
        $data = Docs::query(true)->where('id',$id)->with(['user','docsClass'])->first();
	    [$docs] = $this->docsExtracted();
		[$docsAll] = $this->docsAllExtracted();
	    @$shang = @Docs::query()->where('class_id',$class_id)->where('id','<',$id)->select('title','id')->orderBy('id','desc')->first()?:null;
	    @$xia = @Docs::query()->where('class_id',$class_id)->where('id','>',$id)->select('title','id')->orderBy('id','asc')->first()?:null;
        return @view("Docs::showDocs",['data' => $data,'docs'=>$docs,'docsAll'=>$docsAll,@'shang'=>@$shang,@'xia'=>@$xia]);
    }
	
	/**
	 * @return array
	 */
	public function docsExtracted(): array
	{
		$class = DocsClass::query()->select('id', 'name')->get()->toArray();
		$docs = [];
		foreach($class as $value) {
			$docs[$value['id']]['docs'] = Docs::query()->where('class_id', $value['id'])->take(5)->select('id', 'title', 'class_id')->get();
			$docs[$value['id']]['name'] = $value['name'];
		}
		return array($docs);
	}
	
	/**
	 * @return array
	 */
	public function docsAllExtracted(): array
	{
		$class = DocsClass::query()->select('id', 'name')->get()->toArray();
		$docs = [];
		foreach($class as $value) {
			$docs[$value['id']]['docs'] = Docs::query()->where('class_id', $value['id'])->select('id', 'title', 'class_id')->get();
			$docs[$value['id']]['name'] = $value['name'];
		}
		return array($docs);
	}
	
	
}