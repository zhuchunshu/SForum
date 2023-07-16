<?php

use Hyperf\Database\Schema\Schema;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Migrations\Migration;

class UpdateUsersOptionsCreditsGoldsExpTostringChangeTable extends Migration
{
    /**
     * Run the migrations.
     */
    public function up(): void
    {
        Schema::table('users_options', function (Blueprint $table) {
            $table->string('golds')->comment('金币')->default(0)->change();
            $table->string('credits')->comment('积分')->default(0)->change();
            $table->string('exp')->comment('经验')->default(0)->change();
        });
    }

    /**
     * Reverse the migrations.
     */
    public function down(): void
    {
        Schema::table('users_options', function (Blueprint $table) {
            //
        });
    }
}
