/**
	@file
	@brief C interface of ppzksnark.h
	@author zwj
*/

#ifdef __cplusplus
extern "C" {
#endif

int   r1cs_ppzkadsnark_Generator(char *path, int num_constraints,int num_inputs);
int   r1cs_ppzkSNARK_Prover(char *path);
int   r1cs_ppzkSNARK_Verifier(char  *vk,char   *input, char  *proof);

#ifdef __cplusplus    
}
#endif

