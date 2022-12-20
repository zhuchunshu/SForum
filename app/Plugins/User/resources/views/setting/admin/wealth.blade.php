<div class="card card-body">
    <div class="row">
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.money name")}}
            </label>
            <input type="text" v-model="data.wealth_money_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_money_name','余额')])}}</small>
        </div>

        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.money unit name",['name' => get_options('wealth_money_name','余额')])}}
            </label>
            <input type="text" v-model="data.wealth_money_unit_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_money_unit_name','元')])}}</small>
        </div>

        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.credit name")}}
            </label>
            <input type="text" v-model="data.wealth_credit_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_credit_name','积分')])}}</small>
        </div>
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.golds name")}}
            </label>
            <input type="text" v-model="data.wealth_golds_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_golds_name','金币')])}}</small>
        </div>
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.exp name")}}
            </label>
            <input type="text" v-model="data.wealth_exp_name" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_exp_name','经验')])}}</small>
        </div>
        <div class="col-12">
            <hr class="hr-text">
            <h3 class="card-title">{{__("admin.wealth.exchange rate")}}</h3>
            <div class="alert alert-important alert-danger alert-dismissible" role="alert">
                <div class="d-flex">
                    <div>
                        <!-- Download SVG icon from http://tabler-icons.io/i/alert-circle -->
                        <svg xmlns="http://www.w3.org/2000/svg" class="icon alert-icon" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round"><path stroke="none" d="M0 0h24v24H0z" fill="none"></path><circle cx="12" cy="12" r="9"></circle><line x1="12" y1="8" x2="12" y2="12"></line><line x1="12" y1="16" x2="12.01" y2="16"></line></svg>
                    </div>
                    <div>
                        {{__("admin.wealth.exchange alert")}}
                    </div>
                </div>
                <a class="btn-close btn-close-white" data-bs-dismiss="alert" aria-label="close"></a>
            </div>
        </div>
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.how many",['number' => 1,'first' => get_options('wealth_money_name','余额'),'last' => get_options('wealth_golds_name','金币')])}}
            </label>
            <input type="text" v-model="data.wealth_how_many_money_to_golds" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_how_many_money_to_golds','1')])}}</small>
        </div>
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.how many",['number' => 1,'first' => get_options('wealth_money_name','余额'),'last' => get_options('wealth_credit_name','积分')])}}
            </label>
            <input type="text" v-model="data.wealth_how_many_money_to_credit" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_how_many_money_to_credit',get_options('wealth_how_many_money_to_golds','1')*get_options('wealth_how_many_golds_to_credit',10))])}}</small>
        </div>
        <div class="col-lg-3">
            <label for="" class="form-label">
                {{__("admin.wealth.how many",['number' => 1,'first' => get_options('wealth_golds_name','金币'),'last' => get_options('wealth_credit_name','积分')])}}
            </label>
            <input type="text" v-model="data.wealth_how_many_golds_to_credit" class="form-control">
            <small>{{__("admin.current",['current' => get_options('wealth_how_many_golds_to_credit',10)])}}</small>
        </div>
        <div class="col-lg-3">
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.wealth_close_redemption_money_to_golds">
                <span class="form-check-label">{{__("admin.wealth.close redemption",['first' => get_options('wealth_money_name','余额'),'last' => get_options('wealth_golds_name','金币')])}}</span>
            </label>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.wealth_close_redemption_money_to_credit">
                <span class="form-check-label">{{__("admin.wealth.close redemption",['first' => get_options('wealth_money_name','余额'),'last' => get_options('wealth_credit_name','积分')])}}</span>
            </label>
            <label class="form-check form-switch">
                <input class="form-check-input" type="checkbox" v-model="data.wealth_close_redemption_golds_to_credit">
                <span class="form-check-label">{{__("admin.wealth.close redemption",['first' => get_options('wealth_golds_name','金币'),'last' => get_options('wealth_credit_name','积分')])}}</span>
            </label>
        </div>
    </div>


</div>