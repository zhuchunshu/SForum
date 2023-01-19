@php

                $contacts = [];
                foreach(\App\Plugins\User\src\Models\UsersPm::query()->where(['from_id'=>auth()->id()])->orWhere('to_id',auth()->id())->orderBy('created_at','desc')->get() as $pms){
                    if((int)$pms->from_id !==auth()->id()){
                        $contacts[] = $pms->from_user;
                    }else if((int)$pms->to_id !==auth()->id()){
                        $contacts[] = $pms->to_user;
                    }
                }
                $contacts = array_unique($contacts);
				foreach($contacts as $key=>$value){
					$count = \App\Plugins\User\src\Models\UsersPm::query()->where(['from_id'=>$value->id,'to_id' => auth()->id(),'read' => false])->count();
					$contacts[$key]['msgCount'] = $count;
				}
@endphp
<div class="border-0 card card-body" id="user-pm-container">
    <div class="row row-cards justify-content-center">
        <div class="col-lg-3">
            <div class="border-0 card">
                <div class="card-header">
                    <h3 class="card-title">联系人</h3>
                </div>
                <div class="list-group list-group-flush overflow-auto" style="max-height: 44rem">
                    @foreach($contacts as $contact)
                        <div class="list-group-item">
                            <div class="row">
                                <div class="col-auto">
                                    <a href="/users/pm/{{$contact->id}}">
                                        <span class="avatar"
                                              style="background-image: url({{super_avatar($contact)}})"></span>
                                    </a>
                                </div>
                                <div class="col text-truncate">
                                    <a href="/users/pm/{{$contact->id}}" class="text-body d-block">{{$contact->username}}</a>
                                    <div class="text-muted text-truncate mt-n1"> @if($contact->options->qianming!=='no bio'){{$contact->options->qianming}}@else{{__("user.no bio")}} @endif</div>
                                </div>
                                @if($contact->msgCount>0)
                                <div class="col-auto"><span class="badge badge-pill bg-red">{{$contact->msgCount}}</span></div>
                                @endif
                            </div>
                        </div>
                    @endforeach
                </div>
            </div>
        </div>
        <div class="col-lg-9">
            <div class="border-0 card">
                <div class="card-header">
                    <h3 class="card-title">私信</h3>
                </div>
                <div class="card-body">
                    请选择要私聊的好友
                </div>
            </div>
        </div>
    </div>
</div>
<link rel="stylesheet" href="{{file_hash('plugins/User/css/pm.css')}}">
